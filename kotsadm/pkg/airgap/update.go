package airgap

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/cursor"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kotsadm/pkg/registry"
	"github.com/replicatedhq/kotsadm/pkg/task"
	"github.com/replicatedhq/kotsadm/pkg/version"
)

func UpdateAppFromAirgap(a *app.App, airgapBundle multipart.File) error {
	if err := task.SetTaskStatus("update-download", "Processing package...", "running"); err != nil {
		return errors.Wrap(err, "failed to set tasks status")
	}

	finishedCh := make(chan struct{})
	defer close(finishedCh)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if err := task.UpdateTaskStatusTimestamp("update-download"); err != nil {
					logger.Error(err)
				}
			case <-finishedCh:
				return
			}
		}
	}()

	var finalError error
	defer func() {
		if finalError == nil {
			if err := task.ClearTaskStatus("update-download"); err != nil {
				logger.Error(errors.Wrap(err, "faild to clear update-download task status"))
			}
		} else {
			if err := task.SetTaskStatus("update-download", finalError.Error(), "failed"); err != nil {
				logger.Error(errors.Wrap(err, "faild to set error on update-download task status"))
			}
		}
	}()

	registrySettings, err := registry.GetRegistrySettingsForApp(a.ID)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to get app registry settings")
	}
	cipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to create aes cipher")
	}

	decodedPassword, err := base64.StdEncoding.DecodeString(registrySettings.PasswordEnc)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to decode")
	}

	decryptedPassword, err := cipher.Decrypt([]byte(decodedPassword))
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to decrypt")
	}

	// Some info about the current version
	currentArchivePath, err := version.GetAppVersionArchive(a.ID, a.CurrentSequence)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to get current archive")
	}
	beforeKotsKinds, err := kotsutil.LoadKotsKindsFromPath(currentArchivePath)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to load current kotskinds")
	}

	if beforeKotsKinds.License == nil {
		err := errors.New("no license found in application")
		finalError = err
		return err
	}

	// Start processing the airgap package
	tmpFile, err := ioutil.TempFile("", "kotsadm")
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to create temp file")
	}
	_, err = io.Copy(tmpFile, airgapBundle)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to copy temp airgap")
	}
	defer os.RemoveAll(tmpFile.Name())

	airgapRoot, err := version.ExtractArchiveToTempDirectory(tmpFile.Name())
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to extract archive")
	}

	if err := task.SetTaskStatus("update-download", "Processing app package...", "running"); err != nil {
		finalError = err
		return errors.Wrap(err, "failed to set task status")
	}

	appNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := task.SetTaskStatus("update-download", scanner.Text(), "running"); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	pullOptions := pull.PullOptions{
		LicenseFile:         filepath.Join(currentArchivePath, "upstream", "userdata", "license.yaml"),
		Namespace:           appNamespace,
		ConfigFile:          filepath.Join(currentArchivePath, "upstream", "userdata", "config.yaml"),
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
			Password:   string(decryptedPassword),
		},
	}

	if _, err := pull.Pull(fmt.Sprintf("replicated://%s", beforeKotsKinds.License.Spec.AppSlug), pullOptions); err != nil {
		finalError = err
		return errors.Wrap(err, "failed to pull")
	}

	afterKotsKinds, err := kotsutil.LoadKotsKindsFromPath(currentArchivePath)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to read after kotskinds")
	}

	bc, err := cursor.NewCursor(beforeKotsKinds.Installation.Spec.UpdateCursor)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to create bc")
	}

	ac, err := cursor.NewCursor(afterKotsKinds.Installation.Spec.UpdateCursor)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to create ac")
	}

	if !bc.Comparable(ac) {
		err := errors.Errorf("cannot compare %q and %q", beforeKotsKinds.Installation.Spec.UpdateCursor, afterKotsKinds.Installation.Spec.UpdateCursor)
		finalError = err
		return err
	}

	if !bc.Before(ac) {
		err := errors.Errorf("Version %s (%s) cannot be installed because version %s (%s) is newer", afterKotsKinds.Installation.Spec.VersionLabel, afterKotsKinds.Installation.Spec.UpdateCursor, beforeKotsKinds.Installation.Spec.VersionLabel, beforeKotsKinds.Installation.Spec.UpdateCursor)
		finalError = err
		return err
	}

	// Create the app in the db
	newSequence, err := version.CreateVersion(a.ID, currentArchivePath, "Airgap Upload", a.CurrentSequence)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to create new version")
	}

	// upload to s3
	if err := version.CreateAppVersionArchive(a.ID, newSequence, currentArchivePath); err != nil {
		finalError = err
		return errors.Wrap(err, "failed to upload to s3")
	}

	if err := preflight.Run(a.ID, newSequence, currentArchivePath); err != nil {
		finalError = err
		return errors.Wrap(err, "failed to start preflights")
	}

	return nil
}
