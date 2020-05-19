package updatechecker

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/license"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/task"
	"github.com/replicatedhq/kots/kotsadm/pkg/upstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
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

	appsList, err := app.ListInstalled()
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
	a, err := app.Get(appID)
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

	if a.UpdateCheckerSpec == "@never" || a.UpdateCheckerSpec == "" {
		Stop(a.ID)
		return nil
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

	_, err = job.AddFunc(a.UpdateCheckerSpec, func() {
		logger.Debug("checking updates for app", zap.String("slug", a.Slug))

		availableUpdates, err := CheckForUpdates(a)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to check updates for app %s", a.Slug))
			return
		}

		if availableUpdates > 0 {
			logger.Debug("updates found for app",
				zap.String("slug", a.Slug),
				zap.Int64("available updates", availableUpdates))
		} else {
			logger.Debug("no updates found for app", zap.String("slug", a.Slug))
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
// returns the number of available updates
func CheckForUpdates(a *app.App) (int64, error) {
	currentStatus, err := task.GetTaskStatus("update-download")
	if err != nil {
		return 0, errors.Wrap(err, "failed to get task status")
	}

	if currentStatus == "running" {
		logger.Debug("update-download is already running, not starting a new one")
		return 0, nil
	}

	if err := task.ClearTaskStatus("update-download"); err != nil {
		return 0, errors.Wrap(err, "failed to clear task status")
	}

	// sync license, this method is only called when online
	_, err = license.Sync(a, "")
	if err != nil {
		return 0, errors.Wrap(err, "failed to sync license")
	}

	// reload app because license sync could have created a new release
	a, err = app.Get(a.ID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get app")
	}

	// download the app
	archiveDir, err := version.GetAppVersionArchive(a.ID, a.CurrentSequence)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get app version archive")
	}

	// we need a few objects from the app to check for updates
	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return 0, errors.Wrap(err, "failed to load kotskinds from path")
	}

	getUpdatesOptions := kotspull.GetUpdatesOptions{
		LicenseFile:    filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"),
		CurrentCursor:  kotsKinds.Installation.Spec.UpdateCursor,
		CurrentChannel: kotsKinds.Installation.Spec.ChannelName,
		Silent:         false,
	}

	upstreamURI := ""
	if kotsKinds.License != nil {
		upstreamURI = fmt.Sprintf("replicated://%s", kotsKinds.License.Spec.AppSlug)
	} else if kotsKinds.UnsignedLicense != nil {
		upstreamURI = kotsKinds.UnsignedLicense.Spec.Endpoint
	}

	updates, err := kotspull.GetUpdates(upstreamURI, getUpdatesOptions)
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
		return 0, nil
	}

	availableUpdates := int64(len(updates))

	go func() {
		defer os.RemoveAll(archiveDir)
		for _, update := range updates {
			// the latest version is in archive dir
			if err := upstream.DownloadUpdate(a.ID, archiveDir, update.Cursor); err != nil {
				logger.Error(err)
			}
		}
	}()

	return availableUpdates, nil
}
