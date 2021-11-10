package updatechecker

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/app"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	license "github.com/replicatedhq/kots/pkg/kotsadmlicense"
	upstream "github.com/replicatedhq/kots/pkg/kotsadmupstream"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
	cron "github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// jobs maps app ids to their cron jobs
var jobs = make(map[string]*cron.Cron)
var mtx sync.Mutex

// Start will start the update checker
// the frequency of those update checks are app specific and can be modified by the user
func Start() error {
	logger.Debug("starting update checker")

	appsList, err := store.GetStore().ListInstalledApps()
	if err != nil {
		return errors.Wrap(err, "failed to list installed apps")
	}

	for _, a := range appsList {
		if a.IsAirgap {
			continue
		}
		if err := Configure(a.ID); err != nil {
			logger.Error(errors.Wrapf(err, "failed to configure app %s", a.Slug))
		}
	}

	return nil
}

// Configure will check if the app has scheduled update checks enabled and:
// if enabled, and cron job was NOT found: add a new cron job to check app updates
// if enabled, and a cron job was found, update the existing cron job with the latest cron spec
// if disabled: stop the current running cron job (if exists)
// no-op for airgap applications
func Configure(appID string) error {
	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	if a.IsAirgap {
		return nil
	}

	logger.Debug("configure update checker for app",
		zap.String("slug", a.Slug))

	mtx.Lock()
	defer mtx.Unlock()

	cronSpec := a.UpdateCheckerSpec

	if cronSpec == "@never" || cronSpec == "" {
		Stop(a.ID)
		return nil
	}

	if cronSpec == "@default" {
		// check for updates every 4 hours
		t := time.Now()
		m := t.Minute()
		h := t.Hour() % 4
		cronSpec = fmt.Sprintf("%d %d/4 * * *", m, h)
	}

	job, ok := jobs[a.ID]
	if ok {
		// job already exists, remove entries
		entries := job.Entries()
		for _, entry := range entries {
			job.Remove(entry.ID)
		}
	} else {
		// job does not exist, create a new one
		job = cron.New(cron.WithChain(
			cron.Recover(cron.DefaultLogger),
		))
	}

	jobAppID := a.ID
	jobAppSlug := a.Slug
	jobSemverAutoDeploy := a.SemverAutoDeploy

	_, err = job.AddFunc(cronSpec, func() {
		logger.Debug("checking updates for app", zap.String("slug", jobAppSlug))

		opts := CheckForUpdatesOpts{
			AppID:            jobAppID,
			SemverAutoDeploy: jobSemverAutoDeploy,
		}
		availableUpdates, err := CheckForUpdates(opts)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to check updates for app %s", jobAppSlug))
			return
		}

		if availableUpdates > 0 {
			logger.Debug("updates found for app",
				zap.String("slug", jobAppSlug),
				zap.Int64("available updates", availableUpdates))
		} else {
			logger.Debug("no updates found for app", zap.String("slug", jobAppSlug))
		}
	})
	if err != nil {
		return errors.Wrap(err, "failed to add func")
	}

	job.Start()
	jobs[a.ID] = job

	return nil
}

// Stop will stop a running cron job (if exists) for a specific app
func Stop(appID string) {
	if jobs == nil {
		logger.Debug("no cron jobs found")
		return
	}
	if job, ok := jobs[appID]; ok {
		job.Stop()
	} else {
		logger.Debug("cron job not found for app", zap.String("appID", appID))
	}
}

type CheckForUpdatesOpts struct {
	AppID            string
	Deploy           bool
	SkipPreflights   bool
	IsCLI            bool
	SemverAutoDeploy apptypes.SemverAutoDeploy
}

// CheckForUpdates checks (and downloads) latest updates for a specific app
// if "deploy" is set to true, the latest version/update will be deployed
// returns the number of available updates
func CheckForUpdates(opts CheckForUpdatesOpts) (int64, error) {
	currentStatus, _, err := store.GetStore().GetTaskStatus("update-download")
	if err != nil {
		return 0, errors.Wrap(err, "failed to get task status")
	}

	if currentStatus == "running" {
		logger.Debug("update-download is already running, not starting a new one")
		return 0, nil
	}

	if err := store.GetStore().ClearTaskStatus("update-download"); err != nil {
		return 0, errors.Wrap(err, "failed to clear task status")
	}

	a, err := store.GetStore().GetApp(opts.AppID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get app")
	}

	// sync license, this method is only called when online
	_, _, err = license.Sync(a, "", false)
	if err != nil {
		return 0, errors.Wrap(err, "failed to sync license")
	}

	// reload app because license sync could have created a new release
	a, err = store.GetStore().GetApp(opts.AppID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get app")
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return 0, errors.Wrap(err, "failed to create temp dir")
	}

	removeArchiveDir := true
	defer func() {
		if removeArchiveDir {
			os.RemoveAll(archiveDir)
		}
	}()

	err = store.GetStore().GetAppVersionArchive(a.ID, a.CurrentSequence, archiveDir)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get app version archive")
	}

	// we need a few objects from the app to check for updates
	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return 0, errors.Wrap(err, "failed to load kotskinds from path")
	}

	latestLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get latest license")
	}

	getUpdatesOptions := kotspull.GetUpdatesOptions{
		License:             latestLicense,
		CurrentCursor:       kotsKinds.Installation.Spec.UpdateCursor,
		CurrentChannelID:    kotsKinds.Installation.Spec.ChannelID,
		CurrentChannelName:  kotsKinds.Installation.Spec.ChannelName,
		CurrentVersionLabel: kotsKinds.Installation.Spec.VersionLabel,
		Silent:              false,
		ReportingInfo:       reporting.GetReportingInfo(a.ID),
	}

	// get updates
	updates, err := kotspull.GetUpdates(fmt.Sprintf("replicated://%s", kotsKinds.License.Spec.AppSlug), getUpdatesOptions)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get updates")
	}

	// update last updated at time
	t := app.LastUpdateAtTime(a.ID)
	if t != nil {
		return 0, errors.Wrap(err, "failed to update last updated at time")
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to list downstreams for app")
	}
	if len(downstreams) == 0 {
		return 0, errors.Errorf("no downstreams found for app %q", a.Slug)
	}
	downstream := downstreams[0]

	// if there are updates, go routine it
	if len(updates) == 0 {
		if !opts.Deploy {
			return 0, nil
		}

		// ensure that the latest version is deployed
		allVersions, err := version.GetVersions(a.ID)
		if err != nil {
			return 0, errors.Wrap(err, "failed to list app versions")
		}

		// get the first version, the array must contain versions at this point
		// this function can't run without an app
		if len(allVersions) == 0 {
			return 0, errors.New("no versions found")
		}

		latestVersion := allVersions[len(allVersions)-1]
		downstreamParentSequence, err := store.GetStore().GetCurrentParentSequence(a.ID, downstream.ClusterID)
		if err != nil {
			return 0, errors.Wrap(err, "failed to get current downstream parent sequence")
		}

		if latestVersion.Sequence != downstreamParentSequence {
			status, err := store.GetStore().GetStatusForVersion(a.ID, downstream.ClusterID, latestVersion.Sequence)
			if err != nil {
				return 0, errors.Wrap(err, "failed to get update downstream status")
			}

			if status == storetypes.VersionPendingConfig {
				return 0, util.ActionableError{
					NoRetry: true,
					Message: fmt.Sprintf("Version %d cannot be deployed because it needs configuration", latestVersion.Sequence),
				}
			}

			if err := version.DeployVersion(a.ID, latestVersion.Sequence); err != nil {
				return 0, errors.Wrap(err, "failed to deploy latest version")
			}
		}

		return 0, nil
	}

	availableUpdates := int64(len(updates))

	// this is to avoid a race condition where the UI polls the task status before it is set by the goroutine
	status := fmt.Sprintf("%d Updates available...", availableUpdates)
	if err := store.GetStore().SetTaskStatus("update-download", status, "running"); err != nil {
		return 0, errors.Wrap(err, "failed to set task status")
	}

	removeArchiveDir = false
	go func() {
		defer os.RemoveAll(archiveDir)

		currentVersionLabel := kotsKinds.Installation.Spec.VersionLabel
		indexToDeploy := findUpdateIndexToDeploy(opts, updates, currentVersionLabel)

		for index, update := range updates {
			// the latest version is in archive dir
			sequence, err := upstream.DownloadUpdate(a.ID, archiveDir, update.Cursor, opts.SkipPreflights)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to download update"))
				continue
			}

			if index != indexToDeploy {
				continue
			}

			status, err := store.GetStore().GetStatusForVersion(a.ID, downstream.ClusterID, sequence)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to get update downstream status"))
				continue
			}

			if status == storetypes.VersionPendingConfig {
				logger.Infof("not deploying version %d because it's %s", sequence, status)
				continue
			}

			if err := version.DeployVersion(a.ID, sequence); err != nil {
				logger.Error(errors.Wrap(err, "failed to queue update for deployment"))
			}

			// preflights reporting
			go func() {
				err = reporting.ReportAppInfo(a.ID, sequence, opts.SkipPreflights, opts.IsCLI)
				if err != nil {
					logger.Debugf("failed to update preflights reports: %v", err)
				}
			}()
		}
	}()

	return availableUpdates, nil
}

type UpdateSemverIndex struct {
	Semver semver.Version
	Index  int
}

type UpdateSemverIndicies []UpdateSemverIndex

func (s UpdateSemverIndicies) Len() int { return len(s) }

func (s UpdateSemverIndicies) Less(i, j int) bool {
	return s[i].Semver.Compare(s[j].Semver) == -1
}

func (s UpdateSemverIndicies) Swap(i, j int) {
	tmp := s[i]
	s[i] = s[j]
	s[j] = tmp
}

// findUpdateIndexToDeploy will return the index of the last downloaded update if the "Deploy" option is set to true,
// if not, it will return the index for the update that matches the semver auto deploy configuration (e.g.: latest patch, latest minor, or latest major).
func findUpdateIndexToDeploy(opts CheckForUpdatesOpts, updates []upstreamtypes.Update, currentVersionLabel string) int {
	if opts.Deploy {
		return len(updates) - 1
	}

	if opts.SemverAutoDeploy == "" || opts.SemverAutoDeploy == apptypes.SemverAutoDeployDisabled {
		return -1
	}

	currentSemver, err := semver.ParseTolerant(currentVersionLabel)
	if err != nil {
		return -1
	}

	semverIndicies := UpdateSemverIndicies{}

	for i, update := range updates {
		switch opts.SemverAutoDeploy {
		case apptypes.SemverAutoDeployPatch:
			s, err := semver.ParseTolerant(update.VersionLabel)
			if err != nil {
				continue
			}
			if s.Major != currentSemver.Major || s.Minor != currentSemver.Minor {
				continue
			}
			semverIndicies = append(semverIndicies, UpdateSemverIndex{
				Semver: s,
				Index:  i,
			})

		case apptypes.SemverAutoDeployMinorPatch:
			s, err := semver.ParseTolerant(update.VersionLabel)
			if err != nil {
				continue
			}
			if s.Major != currentSemver.Major {
				continue
			}
			semverIndicies = append(semverIndicies, UpdateSemverIndex{
				Semver: s,
				Index:  i,
			})

		case apptypes.SemverAutoDeployMajorMinorPatch:
			s, err := semver.ParseTolerant(update.VersionLabel)
			if err != nil {
				continue
			}
			semverIndicies = append(semverIndicies, UpdateSemverIndex{
				Semver: s,
				Index:  i,
			})
		}
	}

	if len(semverIndicies) == 0 {
		return -1
	}
	sort.Sort(sort.Reverse(semverIndicies))

	return semverIndicies[len(semverIndicies)-1].Index
}
