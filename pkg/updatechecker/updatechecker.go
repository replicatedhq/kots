package updatechecker

import (
	"fmt"
	"sync"
	"time"

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

	_, err = job.AddFunc(cronSpec, func() {
		logger.Debug("checking updates for app", zap.String("slug", jobAppSlug))

		opts := CheckForUpdatesOpts{
			AppID:       jobAppID,
			IsAutomatic: true,
		}
		ucr, err := CheckForUpdates(opts)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to check updates for app %s", jobAppSlug))
			return
		}

		if ucr.AvailableUpdates > 0 {
			logger.Debug("updates found for app",
				zap.String("slug", jobAppSlug),
				zap.Int64("available updates", ucr.AvailableUpdates))
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
	AppID              string
	DeployLatest       bool
	DeployVersionLabel string
	IsAutomatic        bool
	SkipPreflights     bool
	IsCLI              bool
}

type UpdateCheckResponse struct {
	AvailableUpdates  int64
	CurrentRelease    *UpdateCheckRelease
	AvailableReleases []UpdateCheckRelease
	DeployingRelease  *UpdateCheckRelease
}

type UpdateCheckRelease struct {
	Sequence int64
	Version  string
}

// CheckForUpdates checks, downloads, and makes sure the desired version for a specific app is deployed.
// if "DeployLatest" is set to true, the latest version will be deployed.
// otherwise, if "DeployVersionLabel" is set to true, then the version with the corresponding version label will be deployed (if found).
// otherwise, if "IsAutomatic" is set to true (which means it's an automatic update check), then the version that matches the semver auto deploy configuration (if enabled) will be deployed.
// returns the number of available updates.
func CheckForUpdates(opts CheckForUpdatesOpts) (*UpdateCheckResponse, error) {
	currentStatus, _, err := store.GetStore().GetTaskStatus("update-download")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get task status")
	}

	if currentStatus == "running" {
		logger.Debug("update-download is already running, not starting a new one")
		return nil, nil
	}

	if err := store.GetStore().ClearTaskStatus("update-download"); err != nil {
		return nil, errors.Wrap(err, "failed to clear task status")
	}

	a, err := store.GetStore().GetApp(opts.AppID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}

	// sync license, this method is only called when online
	latestLicense, _, err := license.Sync(a, "", false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sync license")
	}

	// reload app because license sync could have created a new release
	a, err = store.GetStore().GetApp(opts.AppID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}

	updateCursor, versionLabel, err := store.GetStore().GetCurrentUpdateCursor(a.ID, latestLicense.Spec.ChannelID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current update cursor")
	}

	lastUpdateCheckAt, err := time.Parse(time.RFC3339, a.LastUpdateCheckAt)
	if err != nil {
		lastUpdateCheckAt = a.CreatedAt // first time to check for updates, use installation time instead
	}

	getUpdatesOptions := kotspull.GetUpdatesOptions{
		License:             latestLicense,
		LastUpdateCheckAt:   lastUpdateCheckAt,
		CurrentCursor:       updateCursor,
		CurrentChannelID:    latestLicense.Spec.ChannelID,
		CurrentChannelName:  latestLicense.Spec.ChannelName,
		CurrentVersionLabel: versionLabel,
		ChannelChanged:      a.ChannelChanged,
		Silent:              false,
		ReportingInfo:       reporting.GetReportingInfo(a.ID),
	}

	// get updates
	updates, err := kotspull.GetUpdates(fmt.Sprintf("replicated://%s", latestLicense.Spec.AppSlug), getUpdatesOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get updates")
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams for app")
	}
	if len(downstreams) == 0 {
		return nil, errors.Errorf("no downstreams found for app %q", a.Slug)
	}
	d := downstreams[0]

	// get app version labels and sequence numbers
	appVersions, err := store.GetStore().GetAppVersions(opts.AppID, d.ClusterID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get app versions for app %s", opts.AppID)
	}
	if len(appVersions.AllVersions) == 0 {
		return nil, errors.Errorf("no app versions found for app %s in downstream %s", opts.AppID, d.ClusterID)
	}

	var availableReleases []UpdateCheckRelease
	availableSequence := appVersions.AllVersions[0].Sequence + 1
	for _, u := range updates {
		availableReleases = append(availableReleases, UpdateCheckRelease{
			Sequence: availableSequence,
			Version:  u.VersionLabel,
		})
		availableSequence++
	}

	ucr := UpdateCheckResponse{
		AvailableUpdates:  int64(len(updates)),
		AvailableReleases: availableReleases,
		DeployingRelease:  getVersionToDeploy(opts, d.ClusterID, availableReleases),
	}

	if appVersions.CurrentVersion != nil {
		ucr.CurrentRelease = &UpdateCheckRelease{
			Sequence: appVersions.CurrentVersion.Sequence,
			Version:  appVersions.CurrentVersion.VersionLabel,
		}
	}

	if len(updates) == 0 {
		if err := ensureDesiredVersionIsDeployed(opts, d.ClusterID); err != nil {
			return nil, errors.Wrapf(err, "failed to ensure desired version is deployed")
		}
		return &ucr, nil
	}

	// this is to avoid a race condition where the UI polls the task status before it is set by the goroutine
	status := fmt.Sprintf("%d Updates available...", ucr.AvailableUpdates)
	if err := store.GetStore().SetTaskStatus("update-download", status, "running"); err != nil {
		return nil, errors.Wrap(err, "failed to set task status")
	}

	// there are updates, go routine it
	go func() {
		for index, update := range updates {
			_, err = upstream.DownloadUpdate(a.ID, update, opts.SkipPreflights)
			if err != nil {
				logger.Error(errors.Wrapf(err, "failed to download update %s", update.VersionLabel))
				if index == len(updates)-1 {
					// if the last update fails to be downloaded, then the operation isn't successful
					// and lastUpdateCheckTimestamp shouldn't be updated yet since that timestamp is used in detecting new updates
					return
				}
				continue
			}
			// if any update from the channel has been downloaded and processed successfully, then reset the "channel_chaged" flag
			if err = store.GetStore().SetAppChannelChanged(a.ID, false); err != nil {
				logger.Error(errors.Wrapf(err, "failed to reset channel changed flag"))
			}
		}
		if err := app.SetLastUpdateAtTime(a.ID); err != nil {
			logger.Error(errors.Wrap(err, "failed to update last updated at time"))
		}
		if err := ensureDesiredVersionIsDeployed(opts, d.ClusterID); err != nil {
			logger.Error(errors.Wrapf(err, "failed to ensure desired version is deployed"))
		}
	}()

	return &ucr, nil
}

func ensureDesiredVersionIsDeployed(opts CheckForUpdatesOpts, clusterID string) error {
	if opts.DeployLatest {
		if err := deployLatestVersion(opts, clusterID); err != nil {
			return errors.Wrap(err, "failed to deploy latest version")
		}
		return nil
	}

	if opts.DeployVersionLabel != "" {
		if err := deployVersionLabel(opts, clusterID, opts.DeployVersionLabel); err != nil {
			return errors.Wrapf(err, "failed to deploy version label %s", opts.DeployVersionLabel)
		}
		return nil
	}

	if opts.IsAutomatic {
		a, err := store.GetStore().GetApp(opts.AppID)
		if err != nil {
			return errors.Wrap(err, "failed to get app")
		}
		if err := autoDeploy(opts, clusterID, a.SemverAutoDeploy); err != nil {
			return errors.Wrap(err, "failed to auto deploy")
		}
		return nil
	}

	return nil
}

func getVersionToDeploy(opts CheckForUpdatesOpts, clusterID string, availableReleases []UpdateCheckRelease) *UpdateCheckRelease {
	appVersions, err := store.GetStore().GetAppVersions(opts.AppID, clusterID)
	if err != nil {
		return nil
	}
	if len(appVersions.AllVersions) == 0 {
		return nil
	}

	// prepend updates
	for _, u := range availableReleases {
		appVersions.AllVersions = append([]*downstreamtypes.DownstreamVersion{{VersionLabel: u.Version, Sequence: u.Sequence}}, appVersions.AllVersions...)
	}

	if opts.DeployLatest && appVersions.AllVersions[0].Sequence != appVersions.CurrentVersion.Sequence {
		return &UpdateCheckRelease{
			Sequence: appVersions.AllVersions[0].Sequence,
			Version:  appVersions.AllVersions[0].VersionLabel,
		}
	}

	if opts.DeployVersionLabel != "" {
		var versionToDeploy *downstreamtypes.DownstreamVersion
		for _, v := range appVersions.AllVersions {
			if v.VersionLabel == opts.DeployVersionLabel {
				versionToDeploy = v
				break
			}
		}

		if versionToDeploy != nil && versionToDeploy.Sequence != appVersions.CurrentVersion.Sequence {
			return &UpdateCheckRelease{
				Sequence: versionToDeploy.Sequence,
				Version:  versionToDeploy.VersionLabel,
			}
		}
	}

	// todo: get version to deploy for opts.AutoDeploy

	return nil
}

func deployLatestVersion(opts CheckForUpdatesOpts, clusterID string) error {
	appVersions, err := store.GetStore().GetAppVersions(opts.AppID, clusterID)
	if err != nil {
		return errors.Wrapf(err, "failed to get app versions for app %s", opts.AppID)
	}
	if len(appVersions.AllVersions) == 0 {
		return errors.Errorf("no app versions found for app %s in downstream %s", opts.AppID, clusterID)
	}
	latestVersion := appVersions.AllVersions[0]

	if err := deployVersion(opts, clusterID, appVersions, latestVersion); err != nil {
		return errors.Wrapf(err, "failed to deploy sequence %d with version label %s", latestVersion.Sequence, latestVersion.VersionLabel)
	}

	return nil
}

func deployVersionLabel(opts CheckForUpdatesOpts, clusterID string, versionLabel string) error {
	appVersions, err := store.GetStore().GetAppVersions(opts.AppID, clusterID)
	if err != nil {
		return errors.Wrapf(err, "failed to get app versions for app %s", opts.AppID)
	}
	if len(appVersions.AllVersions) == 0 {
		return errors.Errorf("no app versions found for app %s in downstream %s", opts.AppID, clusterID)
	}

	var versionToDeploy *downstreamtypes.DownstreamVersion

	for _, v := range appVersions.AllVersions {
		if v.VersionLabel == versionLabel {
			versionToDeploy = v
			break
		}
	}

	if versionToDeploy == nil {
		return errors.Errorf("version with label %s could not be found", versionLabel)
	}

	if err := deployVersion(opts, clusterID, appVersions, versionToDeploy); err != nil {
		return errors.Wrapf(err, "failed to deploy sequence %d with version label %s", versionToDeploy.Sequence, versionToDeploy.VersionLabel)
	}

	return nil
}

func autoDeploy(opts CheckForUpdatesOpts, clusterID string, semverAutoDeploy apptypes.SemverAutoDeploy) error {
	if semverAutoDeploy == "" || semverAutoDeploy == apptypes.SemverAutoDeployDisabled {
		return nil
	}

	appVersions, err := store.GetStore().GetAppVersions(opts.AppID, clusterID)
	if err != nil {
		return errors.Wrapf(err, "failed to get app versions for app %s", opts.AppID)
	}
	if len(appVersions.AllVersions) == 0 {
		return errors.Errorf("no app versions found for app %s in downstream %s", opts.AppID, clusterID)
	}

	currentVersion := appVersions.CurrentVersion
	if currentVersion == nil || currentVersion.Semver == nil {
		return nil
	}

	var versionToDeploy *downstreamtypes.DownstreamVersion

Loop:
	for _, v := range appVersions.AllVersions {
		if v == nil || v.Semver == nil {
			continue
		}

		if v.Semver.LTE(*currentVersion.Semver) {
			// remaining versions are all gonna have lower semvers
			break
		}

		switch semverAutoDeploy {
		case apptypes.SemverAutoDeployPatch:
			if v.Semver.Major == currentVersion.Semver.Major && v.Semver.Minor == currentVersion.Semver.Minor {
				versionToDeploy = v
				break Loop
			}

		case apptypes.SemverAutoDeployMinorPatch:
			if v.Semver.Major == currentVersion.Semver.Major {
				versionToDeploy = v
				break Loop
			}

		case apptypes.SemverAutoDeployMajorMinorPatch:
			versionToDeploy = v
			break Loop
		}
	}

	if versionToDeploy == nil {
		return nil
	}

	if err := deployVersion(opts, clusterID, appVersions, versionToDeploy); err != nil {
		return errors.Wrapf(err, "failed to deploy sequence %d with version label %s", versionToDeploy.Sequence, versionToDeploy.VersionLabel)
	}

	return nil
}

func deployVersion(opts CheckForUpdatesOpts, clusterID string, appVersions *downstreamtypes.DownstreamVersions, versionToDeploy *downstreamtypes.DownstreamVersion) error {
	if appVersions.CurrentVersion != nil {
		isPastVersion := false
		for _, p := range appVersions.PastVersions {
			if versionToDeploy.Sequence == p.Sequence {
				isPastVersion = true
				break
			}
		}
		if isPastVersion {
			allowRollback, err := store.GetStore().IsRollbackSupportedForVersion(opts.AppID, appVersions.AllVersions[0].Sequence)
			if err != nil {
				return errors.Wrap(err, "failed to check if rollback is supported")
			}
			if !allowRollback {
				return errors.Errorf("version %s is lower than the currently deployed version %s and rollback is not enabled", versionToDeploy.VersionLabel, appVersions.CurrentVersion.VersionLabel)
			}
		}
	}

	downstreamSequence, err := store.GetStore().GetCurrentSequence(opts.AppID, clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current downstream parent sequence")
	}

	if versionToDeploy.Sequence != downstreamSequence {
		status, err := store.GetStore().GetStatusForVersion(opts.AppID, clusterID, versionToDeploy.Sequence)
		if err != nil {
			return errors.Wrap(err, "failed to get update downstream status")
		}

		if status == storetypes.VersionPendingConfig {
			return util.ActionableError{
				NoRetry: true,
				Message: fmt.Sprintf("Version %d cannot be deployed because it needs configuration", versionToDeploy.Sequence),
			}
		}

		if err := version.DeployVersion(opts.AppID, versionToDeploy.Sequence); err != nil {
			return errors.Wrap(err, "failed to deploy version")
		}

		// preflights reporting
		go func() {
			err = reporting.ReportAppInfo(opts.AppID, versionToDeploy.Sequence, opts.SkipPreflights, opts.IsCLI)
			if err != nil {
				logger.Debugf("failed to update preflights reports: %v", err)
			}
		}()
	}

	return nil
}
