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
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
)

func DownloadUpdate(appID string, archiveDir string, toCursor string) (sequence int64, finalError error) {
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
				logger.Error(err)
			}
		} else {
			if err := store.GetStore().SetTaskStatus("update-download", finalError.Error(), "failed"); err != nil {
				logger.Error(err)
			}
		}
	}()

	beforeKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return 0, errors.Wrap(err, "failed to read kots kinds before update")
	}

	beforeCursor := beforeKotsKinds.Installation.Spec.UpdateCursor

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

	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get app")
	}

	appNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	appSequence, err := version.GetNextAppSequence(a.ID, &a.CurrentSequence)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get new app sequence")
	}

	latestLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get latest license")
	}

	pullOptions := kotspull.PullOptions{
		LicenseObj:          latestLicense,
		Namespace:           appNamespace,
		ConfigFile:          filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"),
		InstallationFile:    filepath.Join(archiveDir, "upstream", "userdata", "installation.yaml"),
		UpdateCursor:        toCursor,
		RootDir:             archiveDir,
		ExcludeKotsKinds:    true,
		ExcludeAdminConsole: true,
		CreateAppDir:        false,
		ReportWriter:        pipeWriter,
		AppSlug:             a.Slug,
		AppSequence:         appSequence,
		IsGitOps:            a.IsGitOps,
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(appID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get registry settings")
	}

	if registrySettings != nil {
		pullOptions.RewriteImages = true

		cipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
		if err != nil {
			return 0, errors.Wrap(err, "failed to create aes cipher")
		}

		decodedPassword, err := base64.StdEncoding.DecodeString(registrySettings.PasswordEnc)
		if err != nil {
			return 0, errors.Wrap(err, "failed to decode")
		}

		decryptedPassword, err := cipher.Decrypt([]byte(decodedPassword))
		if err != nil {
			return 0, errors.Wrap(err, "failed to decrypt")
		}

		pullOptions.RewriteImageOptions = kotspull.RewriteImageOptions{
			Host:      registrySettings.Hostname,
			Namespace: registrySettings.Namespace,
			Username:  registrySettings.Username,
			Password:  string(decryptedPassword),
		}
	}

	if _, err := kotspull.Pull(fmt.Sprintf("replicated://%s", beforeKotsKinds.License.Spec.AppSlug), pullOptions); err != nil {
		return 0, errors.Wrap(err, "failed to pull")
	}

	afterKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return 0, errors.Wrap(err, "failed to read kots kinds after update")
	}

	if afterKotsKinds.Installation.Spec.UpdateCursor == beforeCursor {
		return 0, nil // ?
	}

	newSequence, err := version.CreateVersion(appID, archiveDir, "Upstream Update", a.CurrentSequence)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create version")
	}

	if err := preflight.Run(appID, newSequence, a.IsAirgap, archiveDir); err != nil {
		return 0, errors.Wrap(err, "failed to run preflights")
	}

	return newSequence, nil
}
