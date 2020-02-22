package app

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kotsadm/pkg/logger"
)

func DownloadUpdate(a *App, archiveDir string, toCursor string) error {
	finishedCh := make(chan struct{})
	defer close(finishedCh)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if err := UpdateTaskStatusTimestamp("update-download"); err != nil {
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
			if err := ClearTaskStatus("update-download"); err != nil {
				logger.Error(err)
			}
		} else {
			if err := SetTaskStatus("update-download", finalError.Error(), "failed"); err != nil {
				logger.Error(err)
			}
		}
	}()

	beforeKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to read kots kinds before update")
	}

	beforeCursor := beforeKotsKinds.Installation.Spec.UpdateCursor

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := SetTaskStatus("update-download", scanner.Text(), "running"); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	pullOptions := kotspull.PullOptions{
		LicenseFile:         filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"),
		Namespace:           os.Getenv("POD_NAMESPACE"),
		ConfigFile:          filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"),
		InstallationFile:    filepath.Join(archiveDir, "upstream", "userdata", "installation.yaml"),
		UpdateCursor:        toCursor,
		RootDir:             archiveDir,
		ExcludeKotsKinds:    true,
		ExcludeAdminConsole: true,
		CreateAppDir:        false,
		ReportWriter:        pipeWriter,
	}

	if a.RegistrySettings != nil {
		pullOptions.RewriteImages = true

		cipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
		if err != nil {
			return errors.Wrap(err, "failed to create aes cipher")
		}

		decodedPassword, err := base64.StdEncoding.DecodeString(a.RegistrySettings.PasswordEnc)
		if err != nil {
			return errors.Wrap(err, "failed to decode")
		}

		decryptedPassword, err := cipher.Decrypt([]byte(decodedPassword))
		if err != nil {
			return errors.Wrap(err, "failed to decrypt")
		}

		pullOptions.RewriteImageOptions = kotspull.RewriteImageOptions{
			Host:      a.RegistrySettings.Hostname,
			Namespace: a.RegistrySettings.Namespace,
			Username:  a.RegistrySettings.Username,
			Password:  string(decryptedPassword),
		}
	}

	if _, err := kotspull.Pull(fmt.Sprintf("replicated://%s", beforeKotsKinds.License.Spec.AppSlug), pullOptions); err != nil {
		finalError = err
		return errors.Wrap(err, "failed to pull")
	}

	afterKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to read kots kinds after update")
	}

	if afterKotsKinds.Installation.Spec.UpdateCursor == beforeCursor {
		return nil // ?
	}

	newSequence, err := a.CreateVersion(archiveDir, "Upstream Update")
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to create version")
	}

	if err := CreateAppVersionArchive(a.ID, newSequence, archiveDir); err != nil {
		finalError = err
		return errors.Wrap(err, "failed to create app version archive")
	}

	return nil
}
