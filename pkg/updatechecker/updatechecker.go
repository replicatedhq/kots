package updatechecker

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/app"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	license "github.com/replicatedhq/kots/pkg/kotsadmlicense"
	upstream "github.com/replicatedhq/kots/pkg/kotsadmupstream"
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

// CheckForUpdates checks, downloads, and in some cases deploys latest updates for a specific app.
// if "Deploy" is set to true, the latest version/update will be deployed.
// otherwise, if "SemverAutoDeploy" is enabled then the version/update that matches the semver auto deploy configuration will be deployed.
// returns the number of available updates.
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
	latestLicense, _, err := license.Sync(a, "", false)
	if err != nil {
		return 0, errors.Wrap(err, "failed to sync license")
	}

	// reload app because license sync could have created a new release
	a, err = store.GetStore().GetApp(opts.AppID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get app")
	}

	updateCursor, versionLabel, err := store.GetStore().GetCurrentUpdateCursor(a.ID, latestLicense.Spec.ChannelID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get current update cursor")
	}

	getUpdatesOptions := kotspull.GetUpdatesOptions{
		License:             latestLicense,
		CurrentCursor:       updateCursor,
		CurrentChannelID:    latestLicense.Spec.ChannelID,
		CurrentChannelName:  latestLicense.Spec.ChannelName,
		CurrentVersionLabel: versionLabel,
		Silent:              false,
		ReportingInfo:       reporting.GetReportingInfo(a.ID),
	}

	// get updates
	updates, err := kotspull.GetUpdates(fmt.Sprintf("replicated://%s", latestLicense.Spec.AppSlug), getUpdatesOptions)
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
	d := downstreams[0]

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
		downstreamParentSequence, err := store.GetStore().GetCurrentParentSequence(a.ID, d.ClusterID)
		if err != nil {
			return 0, errors.Wrap(err, "failed to get current downstream parent sequence")
		}

		if latestVersion.Sequence != downstreamParentSequence {
			status, err := store.GetStore().GetStatusForVersion(a.ID, d.ClusterID, latestVersion.Sequence)
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

	// there are updates, go routine it
	go func() {
		currentVersion, err := store.GetStore().GetCurrentVersion(a.ID, d.ClusterID)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get current downstream version"))
			return
		}

		currentVersionLabel := ""
		if currentVersion != nil {
			currentVersionLabel = currentVersion.VersionLabel
		}
		indexToDeploy := findUpdateIndexToDeploy(opts, updates, currentVersionLabel)

		for index, update := range updates {
			downloadOptions := DownloadUpdateOptions{
				AppID:          a.ID,
				ClusterID:      d.ClusterID,
				Update:         update,
				SkipPreflights: opts.SkipPreflights,
				Deploy:         index == indexToDeploy,
				IsCLI:          opts.IsCLI,
			}
			if err := downloadUpdate(downloadOptions); err != nil {
				logger.Error(errors.Wrapf(err, "failed to download update %s", update.VersionLabel))
				continue
			}
		}
	}()

	return availableUpdates, nil
}

type DownloadUpdateOptions struct {
	AppID          string
	ClusterID      string
	Update         upstreamtypes.Update
	SkipPreflights bool
	Deploy         bool
	IsCLI          bool
}

func downloadUpdate(opts DownloadUpdateOptions) error {
	baseArchiveDir, err := GetBaseArchiveDirForVersion(opts.AppID, opts.ClusterID, opts.Update.VersionLabel)
	if err != nil {
		return errors.Wrapf(err, "failed to get base archive dir for version %s", opts.Update.VersionLabel)
	}
	defer os.RemoveAll(baseArchiveDir)

	sequence, err := upstream.DownloadUpdate(opts.AppID, baseArchiveDir, opts.Update.Cursor, opts.SkipPreflights)
	if err != nil {
		return errors.Wrap(err, "failed to download update")
	}

	if !opts.Deploy {
		return nil
	}

	status, err := store.GetStore().GetStatusForVersion(opts.AppID, opts.ClusterID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to get update downstream status")
	}

	if status == storetypes.VersionPendingConfig {
		logger.Infof("not deploying version %d because it's %s", sequence, status)
		return nil
	}

	if err := version.DeployVersion(opts.AppID, sequence); err != nil {
		logger.Error(errors.Wrap(err, "failed to queue update for deployment"))
	}

	// preflights reporting
	go func() {
		err = reporting.ReportAppInfo(opts.AppID, sequence, opts.SkipPreflights, opts.IsCLI)
		if err != nil {
			logger.Debugf("failed to update preflights reports: %v", err)
		}
	}()

	return nil
}

// GetBaseArchiveDirForVersion returns the base archive directory for a given version label.
// the base archive directory contains data such as config values.
// caller is responsible for cleaning up the created archive dir.
func GetBaseArchiveDirForVersion(appID string, clusterID string, targetVersionLabel string) (string, error) {
	appVersions, err := store.GetStore().GetAppVersions(appID, clusterID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get app versions for app %s", appID)
	}
	if len(appVersions.AllVersions) == 0 {
		return "", errors.Errorf("no app versions found for app %s in downstream %s", appID, clusterID)
	}

	mockVersion := &downstreamtypes.DownstreamVersion{
		Sequence: math.MaxInt64, // to id the mocked version and be able to retrieve it later
	}

	targetSemver, err := semver.ParseTolerant(targetVersionLabel)
	if err == nil {
		mockVersion.Semver = &targetSemver
	}

	appVersions.AllVersions = append(appVersions.AllVersions, mockVersion)
	downstreamtypes.SortDownstreamVersions(appVersions)

	var baseVersion *downstreamtypes.DownstreamVersion
	for i, v := range appVersions.AllVersions {
		if v.Sequence == math.MaxInt64 {
			// this is our mocked version, base it off of the previous version in the sorted list (if exists).
			if i < len(appVersions.AllVersions)-1 {
				baseVersion = appVersions.AllVersions[i+1]
			}
			// remove the mocked version from the list to not affect what the latest version is in case there's no previous version to use as base.
			appVersions.AllVersions = append(appVersions.AllVersions[:i], appVersions.AllVersions[i+1:]...)
			break
		}
	}

	// if a previous version was not found, base off of the latest version
	if baseVersion == nil {
		baseVersion = appVersions.AllVersions[0]
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	err = store.GetStore().GetAppVersionArchive(appID, baseVersion.ParentSequence, archiveDir)
	if err != nil {
		return "", errors.Wrap(err, "failed to get app version archive")
	}

	return archiveDir, nil
}

type UpdateSemverIndex struct {
	Semver semver.Version
	Index  int
}

type UpdateSemverIndices []UpdateSemverIndex

func (s UpdateSemverIndices) Len() int { return len(s) }

func (s UpdateSemverIndices) Less(i, j int) bool {
	return s[i].Semver.LT(s[j].Semver)
}

func (s UpdateSemverIndices) Swap(i, j int) {
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

	semverIndices := UpdateSemverIndices{}

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
			semverIndices = append(semverIndices, UpdateSemverIndex{
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
			semverIndices = append(semverIndices, UpdateSemverIndex{
				Semver: s,
				Index:  i,
			})

		case apptypes.SemverAutoDeployMajorMinorPatch:
			s, err := semver.ParseTolerant(update.VersionLabel)
			if err != nil {
				continue
			}
			semverIndices = append(semverIndices, UpdateSemverIndex{
				Semver: s,
				Index:  i,
			})
		}
	}

	if len(semverIndices) == 0 {
		return -1
	}
	sort.Sort(sort.Reverse(semverIndices))

	return semverIndices[0].Index
}
