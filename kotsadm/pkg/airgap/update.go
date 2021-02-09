package airgap

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/identity"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/cursor"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/util"
)

func UpdateAppFromAirgap(a *apptypes.App, airgapBundlePath string, deploy bool, skipPreflights bool) (finalError error) {
	finishedCh := make(chan struct{})
	defer close(finishedCh)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if err := store.GetStore().UpdateTaskStatusTimestamp("update-download"); err != nil {
					logger.Error(err)
				}
			case <-finishedCh:
				return
			}
		}
	}()

	defer func() {
		if finalError == nil {
			if err := store.GetStore().ClearTaskStatus("update-download"); err != nil {
				logger.Error(errors.Wrap(err, "failed to clear update-download task status"))
			}
		} else {
			if err := store.GetStore().SetTaskStatus("update-download", finalError.Error(), "failed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set error on update-download task status"))
			}
		}
	}()

	if err := store.GetStore().SetTaskStatus("update-download", "Extracting files...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	airgapRoot, err := version.ExtractArchiveToTempDirectory(airgapBundlePath)
	if err != nil {
		return errors.Wrap(err, "failed to extract archive")
	}
	defer os.RemoveAll(airgapRoot)

	err = UpdateAppFromPath(a, airgapRoot, deploy, skipPreflights)
	return errors.Wrap(err, "failed to update app")
}

func UpdateAppFromPath(a *apptypes.App, airgapRoot string, deploy bool, skipPreflights bool) error {
	if err := store.GetStore().SetTaskStatus("update-download", "Processing package...", "running"); err != nil {
		return errors.Wrap(err, "failed to set tasks status")
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get app registry settings")
	}

	currentArchivePath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(currentArchivePath)

	err = store.GetStore().GetAppVersionArchive(a.ID, a.CurrentSequence, currentArchivePath)
	if err != nil {
		return errors.Wrap(err, "failed to get current archive")
	}
	beforeKotsKinds, err := kotsutil.LoadKotsKindsFromPath(currentArchivePath)
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

	appNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	if err := store.GetStore().SetTaskStatus("update-download", "Creating app version...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	appSequence, err := version.GetNextAppSequence(a.ID, &a.CurrentSequence)
	if err != nil {
		return errors.Wrap(err, "failed to get new app sequence")
	}

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := store.GetStore().SetTaskStatus("update-download", scanner.Text(), "running"); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	// Using license from db instead of upstream bundle because the one in db has not been re-marshalled
	license, err := pull.ParseLicenseFromBytes([]byte(a.License))
	if err != nil {
		return errors.Wrap(err, "failed parse license")
	}

	identityConfigFile := filepath.Join(currentArchivePath, "upstream", "userdata", "identityconfig.yaml")
	if _, err := os.Stat(identityConfigFile); os.IsNotExist(err) {
		file, err := identity.InitAppIdentityConfig(a.Slug, kotsv1beta1.Storage{}, crypto.AESCipher{})
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
		ConfigFile:          filepath.Join(currentArchivePath, "upstream", "userdata", "config.yaml"),
		IdentityConfigFile:  identityConfigFile,
		AirgapRoot:          airgapRoot,
		InstallationFile:    filepath.Join(currentArchivePath, "upstream", "userdata", "installation.yaml"),
		UpdateCursor:        beforeKotsKinds.Installation.Spec.UpdateCursor,
		RootDir:             currentArchivePath,
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
		},
		AppSlug:     a.Slug,
		AppSequence: appSequence,
	}

	if _, err := pull.Pull(fmt.Sprintf("replicated://%s", beforeKotsKinds.License.Spec.AppSlug), pullOptions); err != nil {
		return errors.Wrap(err, "failed to pull")
	}

	afterKotsKinds, err := kotsutil.LoadKotsKindsFromPath(currentArchivePath)
	if err != nil {
		return errors.Wrap(err, "failed to read after kotskinds")
	}

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

	if bc.Equal(ac) {
		return util.ActionableError{Message: fmt.Sprintf("Version %s (%s) cannot be installed again because it is already the current version", afterKotsKinds.Installation.Spec.VersionLabel, afterKotsKinds.Installation.Spec.UpdateCursor)}
	} else if bc.After(ac) {
		return util.ActionableError{Message: fmt.Sprintf("Version %s (%s) cannot be installed because version %s (%s) is newer", afterKotsKinds.Installation.Spec.VersionLabel, afterKotsKinds.Installation.Spec.UpdateCursor, beforeKotsKinds.Installation.Spec.VersionLabel, beforeKotsKinds.Installation.Spec.UpdateCursor)}
	}

	// Create the app in the db
	newSequence, err := store.GetStore().CreateAppVersion(a.ID, &a.CurrentSequence, currentArchivePath, "Airgap Upload", skipPreflights, &version.DownstreamGitOps{})
	if err != nil {
		return errors.Wrap(err, "failed to create new version")
	}

	if !skipPreflights {
		if err := preflight.Run(a.ID, a.Slug, newSequence, true, currentArchivePath); err != nil {
			return errors.Wrap(err, "failed to start preflights")
		}
	}

	if deploy {
		err := version.DeployVersion(a.ID, newSequence)
		if err != nil {
			return errors.Wrap(err, "failed to deploy app version")
		}
	}

	return nil
}
