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
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/rewrite"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
	"go.uber.org/zap"
)

type RegistrySettings struct {
	Hostname    string
	Username    string
	PasswordEnc string
	Namespace   string
}

func UpdateRegistry(appID string, hostname string, username string, password string, namespace string) error {
	logger.Debug("updating app registry",
		zap.String("appID", appID))

	cipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
	if err != nil {
		return errors.Wrap(err, "failed to create aes cipher")
	}

	passwordEnc := base64.StdEncoding.EncodeToString(cipher.Encrypt([]byte(password)))

	db := persistence.MustGetPGSession()
	query := `update app set registry_hostname = $1, registry_username = $2, registry_password_enc = $3, namespace = $4 where id = $5`
	_, err = db.Exec(query, hostname, username, passwordEnc, namespace, appID)
	if err != nil {
		return errors.Wrap(err, "failed to update registry settings")
	}

	return nil
}

// RewriteImages will use the app (a) and send the images to the registry specified. It will create patches for these
// and create a new version of the application
func (a App) RewriteImages(hostname string, username string, password string, namespace string, configValues *kotsv1beta1.ConfigValues) error {
	if err := SetTaskStatus("image-rewrite", "Updating registry settings", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	finishedCh := make(chan struct{})
	defer close(finishedCh)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if err := UpdateTaskStatusTimestamp("image-rewrite"); err != nil {
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
			if err := ClearTaskStatus("image-rewrite"); err != nil {
				logger.Error(err)
			}
		} else {
			if err := SetTaskStatus("image-rewrite", finalError.Error(), "failed"); err != nil {
				logger.Error(err)
			}
		}
	}()

	// get the archive and store it in a temporary location
	appDir, err := GetAppVersionArchive(a.ID, a.CurrentSequence)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to get app version archive")
	}
	defer os.RemoveAll(appDir)

	installation, err := kotsutil.LoadInstallationFromPath(filepath.Join(appDir, "upstream", "userdata", "installation.yaml"))
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to load installation from path")
	}

	license, err := kotsutil.LoadLicenseFromPath(filepath.Join(appDir, "upstream", "userdata", "license.yaml"))
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to load license from path")
	}

	if configValues == nil {
		previousConfigValues, err := kotsutil.LoadConfigValuesFromFile(filepath.Join(appDir, "upstream", "userdata", "config.yaml"))
		if err != nil && !os.IsNotExist(errors.Cause(err)) {
			finalError = err
			return errors.Wrap(err, "failed to load config values from path")
		}

		configValues = previousConfigValues
	}

	// get the downstream names only
	downstreams := []string{}
	for _, downstream := range a.Downstreams {
		downstreams = append(downstreams, downstream.Name)
	}

	// dev_namespace makes the dev env work
	k8sNamespace := "default"
	if os.Getenv("DEV_NAMESPACE") != "" {
		k8sNamespace = os.Getenv("DEV_NAMESPACE")
	}
	if os.Getenv("POD_NAMESPACE") != "" {
		k8sNamespace = os.Getenv("POD_NAMESPACE")
	}

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := SetTaskStatus("image-rewrite", scanner.Text(), "running"); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	options := rewrite.RewriteOptions{
		RootDir:           appDir,
		UpstreamURI:       fmt.Sprintf("replicated://%s", license.Spec.AppSlug),
		UpstreamPath:      filepath.Join(appDir, "upstream"),
		Installation:      installation,
		Downstreams:       downstreams,
		CreateAppDir:      false,
		ExcludeKotsKinds:  true,
		License:           license,
		ConfigValues:      configValues,
		K8sNamespace:      k8sNamespace,
		ReportWriter:      pipeWriter,
		CopyImages:        true,
		IsAirgap:          a.IsAirgap,
		RegistryEndpoint:  hostname,
		RegistryUsername:  username,
		RegistryPassword:  password,
		RegistryNamespace: namespace,
	}

	if err := rewrite.Rewrite(options); err != nil {
		finalError = err
		return errors.Wrap(err, "failed to rewrite images")
	}

	newSequence, err := a.CreateVersion(appDir, "Registry Change")
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to create new version")
	}

	if err := CreateAppVersionArchive(a.ID, newSequence, appDir); err != nil {
		finalError = err
		return errors.Wrap(err, "failed to upload app version")
	}

	return nil
}
