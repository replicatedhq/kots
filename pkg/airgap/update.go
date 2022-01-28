package airgap

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/cursor"
	identity "github.com/replicatedhq/kots/pkg/kotsadmidentity"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
)

func StartUpdateTaskMonitor(finishedChan <-chan error) {
	go func() {
		var finalError error
		defer func() {
			if finalError == nil {
				if err := store.GetStore().ClearTaskStatus("update-download"); err != nil {
					logger.Error(errors.Wrap(err, "failed to clear update-download task status"))
				}
			} else {
				errMsg := finalError.Error()
				if cause, ok := errors.Cause(finalError).(util.ActionableError); ok {
					errMsg = cause.Error()
				}
				if err := store.GetStore().SetTaskStatus("update-download", errMsg, "failed"); err != nil {
					logger.Error(errors.Wrap(err, "failed to set error on update-download task status"))
				}
			}
		}()

		for {
			select {
			case <-time.After(time.Second):
				if err := store.GetStore().UpdateTaskStatusTimestamp("update-download"); err != nil {
					logger.Error(err)
				}
			case err := <-finishedChan:
				finalError = err
				return
			}
		}
	}()
}

func UpdateAppFromAirgap(a *apptypes.App, airgapBundlePath string, deploy bool, skipPreflights bool, skipCompatibilityCheck bool) (finalError error) {
	finishedChan := make(chan error)
	defer close(finishedChan)

	StartUpdateTaskMonitor(finishedChan)
	defer func() {
		finishedChan <- finalError
	}()

	if err := store.GetStore().SetTaskStatus("update-download", "Extracting files...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	airgapRoot, err := extractAppMetaFromAirgapBundle(airgapBundlePath)
	if err != nil {
		return errors.Wrap(err, "failed to extract archive")
	}
	defer os.RemoveAll(airgapRoot)

	err = UpdateAppFromPath(a, airgapRoot, airgapBundlePath, deploy, skipPreflights, skipCompatibilityCheck)
	return errors.Wrap(err, "failed to update app")
}

func UpdateAppFromPath(a *apptypes.App, airgapRoot string, airgapBundlePath string, deploy bool, skipPreflights bool, skipCompatibilityCheck bool) error {
	if err := store.GetStore().SetTaskStatus("update-download", "Processing package...", "running"); err != nil {
		return errors.Wrap(err, "failed to set tasks status")
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get app registry settings")
	}

	airgap, err := pull.FindAirgapMetaInDir(airgapRoot)
	if err != nil {
		return errors.Wrap(err, "failed to parse license from file")
	}

	archiveDir, baseSequence, err := store.GetStore().GetAppVersionBaseArchive(a.ID, airgap.Spec.VersionLabel)
	if err != nil {
		return errors.Wrapf(err, "failed to get base archive dir for version %s", airgap.Spec.VersionLabel)
	}
	defer os.RemoveAll(archiveDir)

	beforeKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return errors.Wrap(err, "failed to load current kotskinds")
	}

	if beforeKotsKinds.License == nil {
		err := errors.New("no license found in application")
		return err
	}

	if err := store.GetStore().SetTaskStatus("update-download", "Processing app package...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	appNamespace := util.AppNamespace()

	if err := store.GetStore().SetTaskStatus("update-download", "Creating app version...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	appSequence, err := store.GetStore().GetNextAppSequence(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get new app sequence")
	}

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := store.GetStore().SetTaskStatus("update-download", scanner.Text(), "running"); err != nil {
				logger.Error(errors.Wrap(err, "failed to update download status"))
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	// Using license from db instead of upstream bundle because the one in db has not been re-marshalled
	license, err := pull.ParseLicenseFromBytes([]byte(a.License))
	if err != nil {
		return errors.Wrap(err, "failed parse license")
	}

	identityConfigFile := filepath.Join(archiveDir, "upstream", "userdata", "identityconfig.yaml")
	if _, err := os.Stat(identityConfigFile); os.IsNotExist(err) {
		file, err := identity.InitAppIdentityConfig(a.Slug, kotsv1beta1.Storage{})
		if err != nil {
			return errors.Wrap(err, "failed to init identity config")
		}
		identityConfigFile = file
		defer os.Remove(identityConfigFile)
	} else if err != nil {
		return errors.Wrap(err, "failed to get stat identity config file")
	}

	pullOptions := pull.PullOptions{
		LicenseObj:          license,
		Namespace:           appNamespace,
		ConfigFile:          filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"),
		IdentityConfigFile:  identityConfigFile,
		AirgapRoot:          airgapRoot,
		AirgapBundle:        airgapBundlePath,
		InstallationFile:    filepath.Join(archiveDir, "upstream", "userdata", "installation.yaml"),
		UpdateCursor:        beforeKotsKinds.Installation.Spec.UpdateCursor,
		RootDir:             archiveDir,
		ExcludeKotsKinds:    true,
		ExcludeAdminConsole: true,
		CreateAppDir:        false,
		ReportWriter:        pipeWriter,
		Silent:              true,
		RewriteImages:       true,
		RewriteImageOptions: pull.RewriteImageOptions{
			ImageFiles: filepath.Join(airgapRoot, "images"),
			Host:       registrySettings.Hostname,
			Namespace:  registrySettings.Namespace,
			Username:   registrySettings.Username,
			Password:   registrySettings.Password,
			IsReadOnly: registrySettings.IsReadOnly,
		},
		AppSlug:     a.Slug,
		AppSequence: appSequence,
	}

	if _, err := pull.Pull(fmt.Sprintf("replicated://%s", beforeKotsKinds.License.Spec.AppSlug), pullOptions); err != nil {
		return errors.Wrap(err, "failed to pull")
	}

	afterKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return errors.Wrap(err, "failed to read after kotskinds")
	}

	if err := canInstall(beforeKotsKinds, afterKotsKinds, skipCompatibilityCheck); err != nil {
		return errors.Wrap(err, "cannot install")
	}

	// Create the app in the db
	newSequence, err := store.GetStore().CreateAppVersion(a.ID, &baseSequence, archiveDir, "Airgap Update", skipPreflights, &version.DownstreamGitOps{})
	if err != nil {
		return errors.Wrap(err, "failed to create new version")
	}

	if !skipPreflights {
		if err := preflight.Run(a.ID, a.Slug, newSequence, true, archiveDir); err != nil {
			return errors.Wrap(err, "failed to start preflights")
		}
	}

	if deploy {
		downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
		if len(downstreams) == 0 {
			return errors.Errorf("no downstreams found for app %q", a.Slug)
		}
		downstream := downstreams[0]

		status, err := store.GetStore().GetStatusForVersion(a.ID, downstream.ClusterID, newSequence)
		if err != nil {
			return errors.Wrap(err, "failed to get update downstream status")
		}

		if status == storetypes.VersionPendingConfig {
			return errors.Errorf("not deploying version %d because it's %s", newSequence, status)
		}

		if err := version.DeployVersion(a.ID, newSequence); err != nil {
			return errors.Wrap(err, "failed to deploy app version")
		}
	}

	return nil
}

func canInstall(beforeKotsKinds *kotsutil.KotsKinds, afterKotsKinds *kotsutil.KotsKinds, skipCompatibilityCheck bool) error {
	if !skipCompatibilityCheck {
		isCompatible, err := kotsutil.IsKotsVersionCompatibleWithApp(afterKotsKinds.KotsApplication, false)
		if err != nil {
			return errors.Wrap(err, "failed to check if kots version is compatible")
		}
		if !isCompatible {
			return util.ActionableError{
				NoRetry: true,
				Message: kotsutil.GetIncompatbileKotsVersionMessage(afterKotsKinds.KotsApplication),
			}
		}
	}

	var beforeSemver, afterSemver *semver.Version
	if v, err := semver.ParseTolerant(beforeKotsKinds.Installation.Spec.VersionLabel); err == nil {
		beforeSemver = &v
	}
	if v, err := semver.ParseTolerant(afterKotsKinds.Installation.Spec.VersionLabel); err == nil {
		afterSemver = &v
	}

	isSameVersion := false

	if beforeSemver != nil && afterSemver != nil {
		// Allow uploading older versions if both have semvers because they can be sorted correctly.
		if beforeSemver.EQ(*afterSemver) {
			isSameVersion = true
		}
	} else if beforeSemver != nil {
		// TODO: pass or fail?
	} else if afterSemver != nil {
		// TODO: pass or fail?
	} else {
		bc, err := cursor.NewCursor(beforeKotsKinds.Installation.Spec.UpdateCursor)
		if err != nil {
			return errors.Wrap(err, "failed to create bc")
		}

		ac, err := cursor.NewCursor(afterKotsKinds.Installation.Spec.UpdateCursor)
		if err != nil {
			return errors.Wrap(err, "failed to create ac")
		}

		if !bc.Comparable(ac) {
			return errors.Errorf("cannot compare %q and %q", beforeKotsKinds.Installation.Spec.UpdateCursor, afterKotsKinds.Installation.Spec.UpdateCursor)
		}

		installChannelID := beforeKotsKinds.Installation.Spec.ChannelID
		licenseChannelID := beforeKotsKinds.License.Spec.ChannelID
		installChannelName := beforeKotsKinds.Installation.Spec.ChannelName
		licenseChannelName := beforeKotsKinds.License.Spec.ChannelName
		if (installChannelID != "" && licenseChannelID != "" && installChannelID == licenseChannelID) || (installChannelName == licenseChannelName) {
			if bc.Equal(ac) {
				isSameVersion = true
			}
		}
	}

	if isSameVersion {
		return util.ActionableError{
			NoRetry: true,
			Message: fmt.Sprintf("Version %s (%s) cannot be installed again because it is already the current version", afterKotsKinds.Installation.Spec.VersionLabel, afterKotsKinds.Installation.Spec.UpdateCursor),
		}
	}

	return nil
}
