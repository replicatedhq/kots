package upstream

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
	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/registry"
	"github.com/replicatedhq/kotsadm/pkg/task"
	"github.com/replicatedhq/kotsadm/pkg/version"
)

func DownloadUpdate(appID string, archiveDir string, toCursor string) error {
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
				logger.Error(err)
			}
		} else {
			if err := task.SetTaskStatus("update-download", finalError.Error(), "failed"); err != nil {
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
			if err := task.SetTaskStatus("update-download", scanner.Text(), "running"); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	appNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("DEV_NAMESPACE") != "" {
		appNamespace = os.Getenv("DEV_NAMESPACE")
	}

	pullOptions := kotspull.PullOptions{
		LicenseFile:         filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"),
		Namespace:           appNamespace,
		ConfigFile:          filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"),
		InstallationFile:    filepath.Join(archiveDir, "upstream", "userdata", "installation.yaml"),
		UpdateCursor:        toCursor,
		RootDir:             archiveDir,
		ExcludeKotsKinds:    true,
		ExcludeAdminConsole: true,
		CreateAppDir:        false,
		ReportWriter:        pipeWriter,
	}

	registrySettings, err := registry.GetRegistrySettingsForApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get registry settings")
	}

	if registrySettings != nil {
		pullOptions.RewriteImages = true

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

		pullOptions.RewriteImageOptions = kotspull.RewriteImageOptions{
			Host:      registrySettings.Hostname,
			Namespace: registrySettings.Namespace,
			Username:  registrySettings.Username,
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

	a, err := app.Get(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	newSequence, err := version.CreateVersion(appID, archiveDir, "Upstream Update", a.CurrentSequence)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to create version")
	}

	if err := version.CreateAppVersionArchive(appID, newSequence, archiveDir); err != nil {
		finalError = err
		return errors.Wrap(err, "failed to create app version archive")
	}

	return nil
}
