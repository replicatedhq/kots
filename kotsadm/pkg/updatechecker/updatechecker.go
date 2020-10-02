package updatechecker

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/k8s"
	"github.com/replicatedhq/kots/kotsadm/pkg/kurl"
	"github.com/replicatedhq/kots/kotsadm/pkg/license"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/upstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
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
	_, err = job.AddFunc(cronSpec, func() {
		logger.Debug("checking updates for app", zap.String("slug", jobAppSlug))

		availableUpdates, err := CheckForUpdates(jobAppID, false)
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

// CheckForUpdates checks (and downloads) latest updates for a specific app
// if "deploy" is set to true, the latest version/update will be deployed
// returns the number of available updates
func CheckForUpdates(appID string, deploy bool) (int64, error) {
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

	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get app")
	}

	// sync license, this method is only called when online
	_, err = license.Sync(a, "", false)
	if err != nil {
		return 0, errors.Wrap(err, "failed to sync license")
	}

	// reload app because license sync could have created a new release
	a, err = store.GetStore().GetApp(appID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get app")
	}

	// download the app
	archiveDir, err := store.GetStore().GetAppVersionArchive(a.ID, a.CurrentSequence)
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
	}

	// add info for reporting purposes
	r, err := GetReportingInfo(a.ID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get reporting info"))
	}
	getUpdatesOptions.ReportingInfo = r

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

	// if there are updates, go routine it
	if len(updates) == 0 {
		if !deploy {
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
		downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
		if err != nil {
			return 0, errors.Wrap(err, "failed to list downstreams for app")
		}

		downstreamParentSequence, err := downstream.GetCurrentParentSequence(a.ID, downstreams[0].ClusterID)
		if err != nil {
			return 0, errors.Wrap(err, "failed to get current downstream parent sequence")
		}

		if latestVersion.Sequence != downstreamParentSequence {
			err := version.DeployVersion(a.ID, latestVersion.Sequence)
			if err != nil {
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

	go func() {
		defer os.RemoveAll(archiveDir)
		for index, update := range updates {
			// the latest version is in archive dir
			sequence, err := upstream.DownloadUpdate(a.ID, archiveDir, update.Cursor)
			if err != nil {
				logger.Error(err)
				continue
			}
			// deploy latest version?
			if deploy && index == len(updates)-1 {
				err := version.DeployVersion(a.ID, sequence)
				if err != nil {
					logger.Error(err)
				}
			}
		}
	}()

	return availableUpdates, nil
}

func GetReportingInfo(appID string) (*upstreamtypes.ReportingInfo, error) {
	r := upstreamtypes.ReportingInfo{
		InstanceID: appID,
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams for app")
	}
	if len(downstreams) == 0 {
		return nil, errors.New("no downstreams found for app")
	}
	r.ClusterID = downstreams[0].ClusterID

	deployedAppSequence, err := downstream.GetCurrentParentSequence(appID, downstreams[0].ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current downstream parent sequence")
	}

	// info about the deployed app sequence
	if deployedAppSequence != -1 {
		deployedArchiveDir, err := store.GetStore().GetAppVersionArchive(appID, deployedAppSequence)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get app version archive")
		}

		deployedKotsKinds, err := kotsutil.LoadKotsKindsFromPath(deployedArchiveDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load kotskinds from path")
		}

		r.DownstreamCursor = deployedKotsKinds.Installation.Spec.UpdateCursor
		r.DownstreamChannelID = deployedKotsKinds.Installation.Spec.ChannelID
		r.DownstreamChannelName = deployedKotsKinds.Installation.Spec.ChannelName
	}

	// get kubernetes cluster version
	clientset, err := k8s.Clientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes clientset")
	}
	k8sVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubernetes server version")
	}
	r.K8sVersion = k8sVersion.GitVersion

	// get app status
	appStatus, err := store.GetStore().GetAppStatus(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app status")
	}
	r.AppStatus = string(appStatus.State)

	// check if embedded cluster
	r.IsKurl = kurl.IsKurl()

	return &r, nil
}
