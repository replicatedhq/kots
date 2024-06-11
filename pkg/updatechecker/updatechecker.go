package updatechecker

import (
	"encoding/json"
	"fmt"
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
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/preflight/types"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/reporting"
	kotssemver "github.com/replicatedhq/kots/pkg/semver"
	storepkg "github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/tasks"
	upstreampkg "github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	cron "github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

// jobs maps app ids to their cron jobs
var jobs = make(map[string]*cron.Cron)
var mtx sync.Mutex
var store = storepkg.GetStore()

// Start will start the update checker
// the frequency of those update checks are app specific and can be modified by the user
func Start() error {
	logger.Debug("starting update checker")

	appsList, err := store.ListInstalledApps()
	if err != nil {
		return errors.Wrap(err, "failed to list installed apps")
	}

	for _, a := range appsList {
		if a.IsAirgap {
			continue
		}
		if err := Configure(a, a.UpdateCheckerSpec); err != nil {
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
func Configure(a *apptypes.App, updateCheckerSpec string) error {
	appId := a.GetID()
	appSlug := a.GetSlug()
	isAirgap := a.GetIsAirgap()

	if isAirgap {
		return nil
	}

	logger.Debug("configure update checker for app",
		zap.String("slug", appSlug))

	mtx.Lock()
	defer mtx.Unlock()

	cronSpec := updateCheckerSpec

	if cronSpec == "@never" || cronSpec == "" {
		Stop(appId)
		return nil
	}

	if cronSpec == "@default" {
		// check for updates every 4 hours
		t := time.Now()
		m := t.Minute()
		h := t.Hour() % 4
		cronSpec = fmt.Sprintf("%d %d/4 * * *", m, h)
	}

	job, ok := jobs[appId]
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

	jobAppID := appId
	jobAppSlug := appSlug

	_, err := job.AddFunc(cronSpec, func() {
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
		if ucr != nil {
			if ucr.AvailableUpdates > 0 {
				logger.Debug("updates found for app",
					zap.String("slug", jobAppSlug),
					zap.Int64("available updates", ucr.AvailableUpdates))
			} else {
				logger.Debug("no updates found for app", zap.String("slug", jobAppSlug))
			}
		}
	})
	if err != nil {
		return errors.Wrap(err, "failed to add func")
	}

	job.Start()
	jobs[appId] = job

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
	AppID                  string
	DeployLatest           bool
	DeployVersionLabel     string
	IsAutomatic            bool
	SkipPreflights         bool
	SkipCompatibilityCheck bool
	IsCLI                  bool
	Wait                   bool
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
// otherwise, if "IsAutomatic" is set to true (which means it's an automatic update check), then the version that matches the auto deploy configuration (if enabled) will be deployed.
// returns the number of available updates.
func CheckForUpdates(opts CheckForUpdatesOpts) (ucr *UpdateCheckResponse, finalError error) {
	currentStatus, _, err := store.GetTaskStatus("update-download")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get task status")
	}
	if currentStatus == "running" {
		logger.Infof("an update check is already running for %s, not starting a new one", opts.AppID)
		return nil, nil
	}

	if err := store.SetTaskStatus("update-download", "Checking for updates...", "running"); err != nil {
		return nil, errors.Wrap(err, "failed to set task status")
	}

	finishedChan := make(chan error, 1)
	defer func() {
		// When "wait" is not set, the go routine will close this channel
		if opts.Wait || finalError != nil || ucr.AvailableUpdates == 0 {
			finishedChan <- finalError
			defer close(finishedChan)
		}
	}()

	tasks.StartUpdateTaskMonitor("update-download", finishedChan)

	ucr, finalError = checkForKotsAppUpdates(opts, finishedChan)
	if finalError != nil {
		finalError = errors.Wrap(finalError, "failed to get kots app updates")
		return
	}

	return
}

func checkForKotsAppUpdates(opts CheckForUpdatesOpts, finishedChan chan<- error) (*UpdateCheckResponse, error) {
	a, err := store.GetApp(opts.AppID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}

	// sync license, this method is only called when online
	latestLicense, _, err := license.Sync(a, "", false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sync license")
	}

	// reload app because license sync could have created a new release
	a, err = store.GetApp(opts.AppID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}

	updateCursor, err := store.GetCurrentUpdateCursor(a.ID, latestLicense.Spec.ChannelID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current update cursor")
	}

	getUpdatesOptions := kotspull.GetUpdatesOptions{
		License:            latestLicense,
		LastUpdateCheckAt:  a.LastUpdateCheckAt,
		CurrentCursor:      updateCursor,
		CurrentChannelID:   latestLicense.Spec.ChannelID,
		CurrentChannelName: latestLicense.Spec.ChannelName,
		ChannelChanged:     a.ChannelChanged,
		Silent:             false,
		ReportingInfo:      reporting.GetReportingInfo(a.ID),
	}

	// get updates
	updates, err := kotspull.GetUpdates(fmt.Sprintf("replicated://%s", latestLicense.Spec.AppSlug), getUpdatesOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get updates")
	}

	downstreams, err := store.ListDownstreamsForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams for app")
	}
	if len(downstreams) == 0 {
		return nil, errors.Errorf("no downstreams found for app %q", a.Slug)
	}
	d := downstreams[0]

	// get app version labels and sequence numbers
	appVersions, err := store.GetDownstreamVersions(opts.AppID, d.ClusterID, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get app versions for app %s", opts.AppID)
	}
	if len(appVersions.AllVersions) == 0 {
		return nil, errors.Errorf("no app versions found for app %s in downstream %s", opts.AppID, d.ClusterID)
	}

	filteredUpdates := removeOldUpdates(updates.Updates, appVersions, latestLicense.Spec.IsSemverRequired)

	var availableReleases []UpdateCheckRelease
	availableSequence := appVersions.AllVersions[0].Sequence + 1
	for _, u := range filteredUpdates {
		availableReleases = append(availableReleases, UpdateCheckRelease{
			Sequence: availableSequence,
			Version:  u.VersionLabel,
		})
		availableSequence++
	}

	ucr := UpdateCheckResponse{
		AvailableUpdates:  int64(len(filteredUpdates)),
		AvailableReleases: availableReleases,
		DeployingRelease:  getVersionToDeploy(opts, d.ClusterID, availableReleases),
	}

	if appVersions.CurrentVersion != nil {
		ucr.CurrentRelease = &UpdateCheckRelease{
			Sequence: appVersions.CurrentVersion.Sequence,
			Version:  appVersions.CurrentVersion.VersionLabel,
		}
	}

	if len(filteredUpdates) == 0 {
		if err := app.SetLastUpdateAtTime(a.ID, updates.UpdateCheckTime); err != nil {
			return nil, errors.Wrap(err, "failed to update last updated at time")
		}
		if err := ensureDesiredVersionIsDeployed(opts, d.ClusterID); err != nil {
			return nil, errors.Wrapf(err, "failed to ensure desired version is deployed")
		}
		return &ucr, nil
	}

	// this is to avoid a race condition where the UI polls the task status before it is set by the goroutine
	status := fmt.Sprintf("%d Updates available...", ucr.AvailableUpdates)
	if err := store.SetTaskStatus("update-download", status, "running"); err != nil {
		return nil, errors.Wrap(err, "failed to set task status")
	}

	if opts.Wait {
		if err := downloadAppUpdates(opts, a.ID, d.ClusterID, filteredUpdates, updates.UpdateCheckTime); err != nil {
			return nil, errors.Wrap(err, "failed to download updates synchronously")
		}
	} else if ucr.AvailableUpdates > 0 {
		go func() {
			defer close(finishedChan)
			err := downloadAppUpdates(opts, a.ID, d.ClusterID, filteredUpdates, updates.UpdateCheckTime)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to download updates asynchronously"))
			}
			finishedChan <- err
		}()
	}

	return &ucr, nil
}

func downloadAppUpdates(opts CheckForUpdatesOpts, appID string, clusterID string, updates []upstreamtypes.Update, updateCheckTime time.Time) error {
	for index, update := range updates {
		appSequence, err := upstream.DownloadUpdate(appID, update, opts.SkipPreflights, opts.SkipCompatibilityCheck)
		if appSequence != nil {
			// a version has been created, reset the "channel_changed" flag regardless if there was an error or not
			if err := store.SetAppChannelChanged(appID, false); err != nil {
				logger.Error(errors.Wrapf(err, "failed to reset channel changed flag"))
			}
		}
		if err != nil {
			err := errors.Wrapf(err, "failed to download update %s", update.VersionLabel)
			if index == len(updates)-1 {
				// if the last update fails to be downloaded, then the operation isn't successful
				// and lastUpdateCheckTimestamp shouldn't be updated yet since that timestamp is used in detecting new updates
				return err
			}
			logger.Error(err)
		}
	}
	if err := app.SetLastUpdateAtTime(appID, updateCheckTime); err != nil {
		return errors.Wrap(err, "failed to update last updated at time")
	}
	if err := ensureDesiredVersionIsDeployed(opts, clusterID); err != nil {
		return errors.Wrapf(err, "failed to ensure desired version is deployed")
	}
	return nil
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

	if !opts.IsAutomatic {
		return nil
	}

	a, err := store.GetApp(opts.AppID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}
	if err := autoDeploy(opts, clusterID, a.AutoDeploy); err != nil {
		return errors.Wrap(err, "failed to auto deploy")
	}
	return nil
}

func getVersionToDeploy(opts CheckForUpdatesOpts, clusterID string, availableReleases []UpdateCheckRelease) *UpdateCheckRelease {
	appVersions, err := store.GetDownstreamVersions(opts.AppID, clusterID, true)
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
	appVersions, err := store.GetDownstreamVersions(opts.AppID, clusterID, true)
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
	appVersions, err := store.GetDownstreamVersions(opts.AppID, clusterID, true)
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

func autoDeploy(opts CheckForUpdatesOpts, clusterID string, autoDeploy apptypes.AutoDeploy) error {
	if autoDeploy == "" || autoDeploy == apptypes.AutoDeployDisabled {
		return nil
	}

	appVersions, err := store.GetDownstreamVersions(opts.AppID, clusterID, true)
	if err != nil {
		return errors.Wrapf(err, "failed to get app versions for app %s", opts.AppID)
	}
	if len(appVersions.AllVersions) == 0 {
		return errors.Errorf("no app versions found for app %s in downstream %s", opts.AppID, clusterID)
	}

	currentVersion := appVersions.CurrentVersion
	if currentVersion == nil {
		return nil
	}

	var versionToDeploy *downstreamtypes.DownstreamVersion

	if autoDeploy == apptypes.AutoDeploySequence {
		// semver is not required/enabled, we only need to check if the newest app version is newer than the current version.
		// use cursor instead of sequence in order to only deploy newer upstream versions, and not versions created by config changes, license changes, etc...
		currentCursor := currentVersion.Cursor
		latestCursor := appVersions.AllVersions[0].Cursor
		if currentCursor != nil && latestCursor != nil && (*currentCursor).Before(*latestCursor) {
			versionToDeploy = appVersions.AllVersions[0]
		} else {
			return nil
		}
	} else if currentVersion.Semver != nil { // semver is required
	Loop:
		for _, v := range appVersions.AllVersions {
			if v == nil || v.Semver == nil {
				continue
			}

			if v.Semver.LTE(*currentVersion.Semver) {
				// remaining versions are all gonna have lower semvers
				break
			}

			switch autoDeploy {
			case apptypes.AutoDeploySemverPatch:
				if v.Semver.Major == currentVersion.Semver.Major && v.Semver.Minor == currentVersion.Semver.Minor {
					versionToDeploy = v
					break Loop
				}

			case apptypes.AutoDeploySemverMinorPatch:
				if v.Semver.Major == currentVersion.Semver.Major {
					versionToDeploy = v
					break Loop
				}

			case apptypes.AutoDeploySemverMajorMinorPatch:
				versionToDeploy = v
				break Loop
			}
		}
	}

	if versionToDeploy == nil {
		return nil
	}

	if err := waitForPreflightsToFinish(opts.AppID, versionToDeploy.Sequence); err != nil {
		return errors.Wrap(err, "not able to auto-deploy due to failed preflight check")
	}

	if err := deployVersion(opts, clusterID, appVersions, versionToDeploy); err != nil {
		return errors.Wrapf(err, "failed to deploy sequence %d with version label %s", versionToDeploy.Sequence, versionToDeploy.VersionLabel)
	}

	return nil
}

func waitForPreflightsToFinish(appID string, sequence int64) error {
	app, err := store.GetApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed get app to check for preflights")
	}

	if !app.HasPreflight {
		return nil
	}

	err = wait.PollImmediate(2*time.Second, 15*time.Minute, func() (bool, error) {
		versionStatus, err := store.GetDownstreamVersionStatus(appID, sequence)
		if err != nil {
			return false, errors.Wrap(err, "failed get status")
		}
		if versionStatus != storetypes.VersionPendingPreflight {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to poll for preflights results")
	}
	// refetch latest results
	preflightResult, err := store.GetPreflightResults(appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to fetch preflight results")
	}

	if preflightResult == nil || len(preflightResult.Result) == 0 {
		return errors.New("failed to find a preflight spec")
	}

	var preflightResults *types.PreflightResults
	if err = json.Unmarshal([]byte(preflightResult.Result), &preflightResults); err != nil {
		return errors.Wrap(err, "failed to parse preflight results")
	}

	state := preflight.GetPreflightState(preflightResults)
	if state == "fail" {
		return errors.New(fmt.Sprintf("errors in the preflight state results: %v", preflightResults))
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
			allowRollback, err := store.IsRollbackSupportedForVersion(opts.AppID, appVersions.AllVersions[0].Sequence)
			if err != nil {
				return errors.Wrap(err, "failed to check if rollback is supported")
			}
			if !allowRollback {
				return errors.Errorf("version %s is lower than the currently deployed version %s and rollback is not enabled", versionToDeploy.VersionLabel, appVersions.CurrentVersion.VersionLabel)
			}
		}
	}

	downstreamSequence, err := store.GetCurrentDownstreamSequence(opts.AppID, clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current downstream parent sequence")
	}

	if versionToDeploy.Sequence != downstreamSequence {
		status, err := store.GetStatusForVersion(opts.AppID, clusterID, versionToDeploy.Sequence)
		if err != nil {
			return errors.Wrapf(err, "failed to get status for version %d", versionToDeploy.Sequence)
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
			err = reporting.WaitAndReportPreflightChecks(opts.AppID, versionToDeploy.Sequence, opts.SkipPreflights, opts.IsCLI)
			if err != nil {
				logger.Debugf("failed to update preflights reports: %v", err)
			}
		}()
	}

	return nil
}

type sortableUpdate struct {
	Sequence       int64
	Semver         *semver.Version
	UpstreamUpdate *upstreamtypes.Update
}

type bySemver []*sortableUpdate

func (v bySemver) Len() int {
	return len(v)
}

func (v bySemver) HasSemver(i int) bool {
	return v[i].Semver != nil
}

func (v bySemver) GetSemver(i int) *semver.Version {
	return v[i].Semver
}

func (v bySemver) GetSequence(i int) int64 {
	return v[i].Sequence
}

func (v bySemver) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

// Removes updates that are older than the first release installed in the cluster
func removeOldUpdates(updates []upstreamtypes.Update, appVersions *downstreamtypes.DownstreamVersions, isSemverRequired bool) []upstreamtypes.Update {
	if !isSemverRequired {
		return updates
	}

	newMaxSequence := appVersions.AllVersions[0].Sequence + int64(len(updates))
	sortedUpdates := []*sortableUpdate{}
	for i := range updates {
		u := updates[i]
		su := &sortableUpdate{
			Sequence:       newMaxSequence,
			UpstreamUpdate: &u,
		}
		if v, err := semver.ParseTolerant(u.VersionLabel); err == nil {
			su.Semver = &v
		}
		sortedUpdates = append(sortedUpdates, su)
		newMaxSequence -= 1 // sorted order is descending
	}
	for i := range appVersions.AllVersions {
		u := appVersions.AllVersions[i]
		su := &sortableUpdate{
			Sequence: u.Sequence,
			Semver:   u.Semver,
		}
		sortedUpdates = append(sortedUpdates, su)
	}

	kotssemver.SortVersions(bySemver(sortedUpdates))

	fileteredUpdates := []upstreamtypes.Update{}
	for _, su := range sortedUpdates {
		if su.Sequence == 0 {
			break
		}
		if su.UpstreamUpdate == nil {
			continue
		}
		fileteredUpdates = append(fileteredUpdates, *su.UpstreamUpdate)
	}

	return fileteredUpdates
}

func GetAvailableUpdates(kotsStore storepkg.Store, app *apptypes.App, license *kotsv1beta1.License) ([]*downstreamtypes.DownstreamVersion, error) {
	updateCursor, err := kotsStore.GetCurrentUpdateCursor(app.ID, license.Spec.ChannelID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current update cursor")
	}

	upstreamURI := fmt.Sprintf("replicated://%s", license.Spec.AppSlug)
	fetchOptions := &upstreamtypes.FetchOptions{
		License:            license,
		LastUpdateCheckAt:  app.LastUpdateCheckAt,
		CurrentCursor:      updateCursor,
		CurrentChannelID:   license.Spec.ChannelID,
		CurrentChannelName: license.Spec.ChannelName,
		ChannelChanged:     app.ChannelChanged,
		SortOrder:          "desc", // get the latest updates first
		ReportingInfo:      reporting.GetReportingInfo(app.ID),
	}
	updates, err := upstreampkg.GetUpdatesUpstream(upstreamURI, fetchOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get updates")
	}

	availableVersions := []*downstreamtypes.DownstreamVersion{}
	for _, u := range updates.Updates {
		availableVersions = append(availableVersions, &downstreamtypes.DownstreamVersion{
			VersionLabel:       u.VersionLabel,
			UpdateCursor:       u.Cursor,
			ChannelID:          u.ChannelID,
			IsRequired:         u.IsRequired,
			Status:             storetypes.VersionPendingDownload,
			UpstreamReleasedAt: u.ReleasedAt,
			ReleaseNotes:       u.ReleaseNotes,
		})
	}

	return availableVersions, nil
}
