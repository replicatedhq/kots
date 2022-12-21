package upstream

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	identity "github.com/replicatedhq/kots/pkg/kotsadmidentity"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/upstream"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
)

func DownloadUpdate(appID string, update types.Update, skipPreflights bool, skipCompatibilityCheck bool) (finalSequence *int64, finalError error) {
	taskID := "update-download"
	var finishedCh chan struct{}
	if update.AppSequence != nil {
		taskID = fmt.Sprintf("update-download.%d", *update.AppSequence)

		// The entire "update-download" task state is managed ouside of this function.
		// Version specific tasks are managed in this sope only.
		finishedCh = make(chan struct{}, 1)
		go func() {
			for {
				select {
				case <-time.After(time.Second):
					if err := store.GetStore().UpdateTaskStatusTimestamp(taskID); err != nil {
						logger.Error(errors.Wrapf(err, "failed to update %s task status timestamp", taskID))
					}
				case <-finishedCh:
					return
				}
			}
		}()
	}

	if err := store.GetStore().SetTaskStatus(taskID, "Fetching update...", "running"); err != nil {
		finalError = errors.Wrap(err, "failed to set task status")
		return
	}

	defer func() {
		if finishedCh != nil {
			close(finishedCh)
		}

		if finalError == nil {
			if update.AppSequence != nil {
				// this could be an older version that is being downloaded at a later point
				// update the diff summary of the next version in the list (if exists)
				err := store.GetStore().UpdateNextAppVersionDiffSummary(appID, *update.AppSequence)
				if err != nil {
					logger.Error(errors.Wrapf(err, "failed to update next app version diff summary for base sequence %d", *update.AppSequence))
				}
			}
			err := store.GetStore().ClearTaskStatus(taskID)
			if err != nil {
				logger.Error(errors.Wrapf(err, "failed to clear %s task status", taskID))
			}
			return
		}

		errMsg := finalError.Error()
		if cause, ok := errors.Cause(finalError).(util.ActionableError); ok {
			errMsg = cause.Error()
		}

		var kotsApplication *kotsv1beta1.Application
		var license *kotsv1beta1.License
		if cause, ok := errors.Cause(finalError).(upstream.IncompatibleAppError); ok {
			errMsg = cause.Error()
			kotsApplication = cause.KotsApplication
			license = cause.License
			finalError = util.ActionableError{
				NoRetry: true,
				Message: cause.Error(),
			}
		}

		if update.AppSequence != nil || finalSequence != nil {
			// a version already exists or has been created
			err := store.GetStore().SetTaskStatus(taskID, errMsg, "failed")
			if err != nil {
				logger.Error(errors.Wrapf(err, "failed to set %s task status", taskID))
			}
			return
		}

		// no version has been created for the update yet, create the version as pending download
		newSequence, err := store.GetStore().CreatePendingDownloadAppVersion(appID, update, kotsApplication, license)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to create pending download app version for update %s", update.VersionLabel))
			if err := store.GetStore().SetTaskStatus(taskID, errMsg, "failed"); err != nil {
				logger.Error(errors.Wrapf(err, "failed to set %s task status", taskID))
			}
			return
		}
		finalSequence = &newSequence

		// a pending download version has been created, bind the download error to it
		// clear the global task status at the end to avoid a race condition with the UI
		sequenceTaskID := fmt.Sprintf("update-download.%d", *finalSequence)
		if err := store.GetStore().SetTaskStatus(sequenceTaskID, errMsg, "failed"); err != nil {
			logger.Error(errors.Wrapf(err, "failed to set %s task status", sequenceTaskID))
		}
		if err := store.GetStore().ClearTaskStatus(taskID); err != nil {
			logger.Error(errors.Wrapf(err, "failed to clear %s task status", taskID))
		}
	}()

	archiveDir, baseSequence, err := store.GetStore().GetAppVersionBaseArchive(appID, update.VersionLabel)
	if err != nil {
		finalError = errors.Wrapf(err, "failed to get base archive dir for version %s", update.VersionLabel)
		return
	}
	defer os.RemoveAll(archiveDir)

	beforeKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		finalError = errors.Wrap(err, "failed to read kots kinds before update")
		return
	}

	beforeCursor := beforeKotsKinds.Installation.Spec.UpdateCursor

	pipeReader, pipeWriter := io.Pipe()
	defer func() {
		pipeWriter.CloseWithError(finalError)
	}()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := store.GetStore().SetTaskStatus(taskID, scanner.Text(), "running"); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		finalError = errors.Wrap(err, "failed to get app")
		return
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		finalError = errors.Wrap(err, "failed to list downstreams for app")
		return
	}

	downstreamNames := []string{}
	for _, d := range downstreams {
		downstreamNames = append(downstreamNames, d.Name)
	}

	appNamespace := util.AppNamespace()

	appSequence, err := store.GetStore().GetNextAppSequence(a.ID)
	if err != nil {
		finalError = errors.Wrap(err, "failed to get new app sequence")
		return
	}
	if update.AppSequence != nil {
		appSequence = *update.AppSequence
	}

	latestLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		finalError = errors.Wrap(err, "failed to get latest license")
		return
	}

	identityConfigFile := filepath.Join(archiveDir, "upstream", "userdata", "identityconfig.yaml")
	if _, err := os.Stat(identityConfigFile); os.IsNotExist(err) {
		file, err := identity.InitAppIdentityConfig(a.Slug)
		if err != nil {
			finalError = errors.Wrap(err, "failed to init identity config")
			return
		}
		identityConfigFile = file
		defer os.Remove(identityConfigFile)
	} else if err != nil {
		finalError = errors.Wrap(err, "failed to get stat identity config file")
		return
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(appID)
	if err != nil {
		finalError = errors.Wrap(err, "failed to get registry settings")
		return
	}

	if err := pull.CleanBaseArchive(archiveDir); err != nil {
		finalError = errors.Wrap(err, "failed to clean base archive")
		return
	}

	pullOptions := pull.PullOptions{
		LicenseObj:             latestLicense,
		Namespace:              appNamespace,
		ConfigFile:             filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"),
		IdentityConfigFile:     identityConfigFile,
		InstallationFile:       filepath.Join(archiveDir, "upstream", "userdata", "installation.yaml"),
		UpdateCursor:           update.Cursor,
		RootDir:                archiveDir,
		Downstreams:            downstreamNames,
		ExcludeKotsKinds:       true,
		ExcludeAdminConsole:    true,
		CreateAppDir:           false,
		ReportWriter:           pipeWriter,
		AppSlug:                a.Slug,
		AppSequence:            appSequence,
		IsGitOps:               a.IsGitOps,
		ReportingInfo:          reporting.GetReportingInfo(a.ID),
		RewriteImages:          registrySettings.IsValid(),
		RewriteImageOptions:    registrySettings,
		SkipCompatibilityCheck: skipCompatibilityCheck,
	}

	_, err = pull.Pull(fmt.Sprintf("replicated://%s", beforeKotsKinds.License.Spec.AppSlug), pullOptions)
	if err != nil {
		if errors.Cause(err) != pull.ErrConfigNeeded {
			finalError = errors.Wrap(err, "failed to pull")
			return
		}
	}

	if update.AppSequence == nil {
		afterKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
		if err != nil {
			finalError = errors.Wrap(err, "failed to read kots kinds after update")
			return
		}
		if afterKotsKinds.Installation.Spec.UpdateCursor == beforeCursor {
			return
		}
		newSequence, err := store.GetStore().CreateAppVersion(a.ID, &baseSequence, archiveDir, "Upstream Update", skipPreflights, &version.DownstreamGitOps{}, render.Renderer{})
		if err != nil {
			finalError = errors.Wrap(err, "failed to create version")
			return
		}
		finalSequence = &newSequence
	} else {
		err := store.GetStore().UpdateAppVersion(a.ID, *update.AppSequence, &baseSequence, archiveDir, "Upstream Update", skipPreflights, &version.DownstreamGitOps{}, render.Renderer{})
		if err != nil {
			finalError = errors.Wrap(err, "failed to create version")
			return
		}
		finalSequence = update.AppSequence
	}

	hasStrictPreflights, err := store.GetStore().HasStrictPreflights(a.ID, *finalSequence)
	if err != nil {
		finalError = errors.Wrap(err, "failed to check if app preflight has strict analyzers")
		return
	}

	if hasStrictPreflights && skipPreflights {
		logger.Warnf("preflights will not be skipped, strict preflights are set to %t", hasStrictPreflights)
	}

	if !skipPreflights || hasStrictPreflights {
		if err := preflight.Run(appID, a.Slug, *finalSequence, a.IsAirgap, archiveDir); err != nil {
			finalError = errors.Wrap(err, "failed to run preflights")
			return
		}
	}

	return
}
