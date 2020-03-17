package registry

import (
	"bufio"
	"database/sql"
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
	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kotsadm/pkg/registry/types"
	"github.com/replicatedhq/kotsadm/pkg/task"
	"github.com/replicatedhq/kotsadm/pkg/version"
	"go.uber.org/zap"
)

func GetRegistrySettingsForApp(appID string) (*types.RegistrySettings, error) {
	db := persistence.MustGetPGSession()
	query := `select registry_hostname, registry_username, registry_password_enc, namespace from app where id = $1`
	row := db.QueryRow(query, appID)

	var registryHostname sql.NullString
	var registryUsername sql.NullString
	var registryPasswordEnc sql.NullString
	var registryNamespace sql.NullString

	if err := row.Scan(&registryHostname, &registryUsername, &registryPasswordEnc, &registryNamespace); err != nil {
		return nil, errors.Wrap(err, "failed to scan registry")
	}

	if !registryHostname.Valid {
		return nil, nil
	}

	registrySettings := types.RegistrySettings{
		Hostname:    registryHostname.String,
		Username:    registryUsername.String,
		PasswordEnc: registryPasswordEnc.String,
		Namespace:   registryNamespace.String,
	}

	return &registrySettings, nil
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
func RewriteImages(appID string, sequence int64, hostname string, username string, password string, namespace string, configValues *kotsv1beta1.ConfigValues) error {
	if err := task.SetTaskStatus("image-rewrite", "Updating registry settings", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	finishedCh := make(chan struct{})
	defer close(finishedCh)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if err := task.UpdateTaskStatusTimestamp("image-rewrite"); err != nil {
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
			if err := task.ClearTaskStatus("image-rewrite"); err != nil {
				logger.Error(err)
			}
		} else {
			if err := task.SetTaskStatus("image-rewrite", finalError.Error(), "failed"); err != nil {
				logger.Error(err)
			}
		}
	}()

	// get the archive and store it in a temporary location
	appDir, err := version.GetAppVersionArchive(appID, sequence)
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
	downstreams, err := downstream.ListDownstreamsForApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to list downstreams")
	}

	downstreamNames := []string{}
	for _, d := range downstreams {
		downstreamNames = append(downstreamNames, d.Name)
	}

	a, err := app.Get(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
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
			if err := task.SetTaskStatus("image-rewrite", scanner.Text(), "running"); err != nil {
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
		Downstreams:       downstreamNames,
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

	newSequence, err := version.CreateVersion(appID, appDir, "Registry Change", a.CurrentSequence)
	if err != nil {
		finalError = err
		return errors.Wrap(err, "failed to create new version")
	}

	if err := version.CreateAppVersionArchive(appID, newSequence, appDir); err != nil {
		finalError = err
		return errors.Wrap(err, "failed to upload app version")
	}

	if err := preflight.Run(appID, newSequence, appDir); err != nil {
		finalError = err
		return errors.Wrap(err, "failed to run preflights")
	}

	return nil
}
