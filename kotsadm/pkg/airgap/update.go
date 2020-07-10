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
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/registry"
	"github.com/replicatedhq/kots/kotsadm/pkg/task"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/cursor"
	"github.com/replicatedhq/kots/pkg/pull"
)

func UpdateAppFromAirgap(a *app.App, airgapBundle multipart.File) (finalError error) {
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

	defer func() {
		if finalError == nil {
			if err := task.ClearTaskStatus("update-download"); err != nil {
				logger.Error(errors.Wrap(err, "failed to clear update-download task status"))
			}
		} else {
			if err := task.SetTaskStatus("update-download", finalError.Error(), "failed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set error on update-download task status"))
			}
		}
	}()

	registrySettings, err := registry.GetRegistrySettingsForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get app registry settings")
	}
	cipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
	if err != nil {
		return errors.Wrap(err, "failed to create aes cipher")
	}

	decodedPassword, err := base64.StdEncoding.DecodeString(registrySettings.PasswordEnc)
	if err != nil {
		return errors.Wrap(err, "failed to decode")
	}

	decryptedPassword, err := cipher.Decrypt([]byte(decodedPassword))
	if err != nil {
		return errors.Wrap(err, "failed to decrypt")
	}

	// Some info about the current version
	currentArchivePath, err := version.GetAppVersionArchive(a.ID, a.CurrentSequence)
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

	// Start processing the airgap package
	tmpFile, err := ioutil.TempFile("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}

	if err := task.SetTaskStatus("update-download", "Copying package...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	_, err = io.Copy(tmpFile, airgapBundle)
	if err != nil {
		return errors.Wrap(err, "failed to copy temp airgap")
	}
	defer os.RemoveAll(tmpFile.Name())

	if err := task.SetTaskStatus("update-download", "Extracting files...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	airgapRoot, err := version.ExtractArchiveToTempDirectory(tmpFile.Name())
	if err != nil {
		return errors.Wrap(err, "failed to extract archive")
	}
	defer os.RemoveAll(airgapRoot)

	if err := task.SetTaskStatus("update-download", "Processing app package...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	appNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	if err := task.SetTaskStatus("update-download", "Creating app version...", "running"); err != nil {
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

	if !bc.Before(ac) {
		return errors.Errorf("Version %s (%s) cannot be installed because version %s (%s) is newer", afterKotsKinds.Installation.Spec.VersionLabel, afterKotsKinds.Installation.Spec.UpdateCursor, beforeKotsKinds.Installation.Spec.VersionLabel, beforeKotsKinds.Installation.Spec.UpdateCursor)
	}

	// Create the app in the db
	newSequence, err := version.CreateVersion(a.ID, currentArchivePath, "Airgap Upload", a.CurrentSequence)
	if err != nil {
		return errors.Wrap(err, "failed to create new version")
	}

	// upload to s3
	if err := version.CreateAppVersionArchive(a.ID, newSequence, currentArchivePath); err != nil {
		return errors.Wrap(err, "failed to upload to s3")
	}

	if err := preflight.Run(a.ID, newSequence, currentArchivePath); err != nil {
		return errors.Wrap(err, "failed to start preflights")
	}

	return nil
}
