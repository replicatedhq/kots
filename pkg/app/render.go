package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/rewrite"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
)

// RenderDir renders an app archive dir
// this is useful for when the license/config have updated, and template functions need to be evaluated again
func (a *App) RenderDir(archiveDir string) error {
	installation, err := kotsutil.LoadInstallationFromPath(filepath.Join(archiveDir, "upstream", "userdata", "installation.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to load installation from path")
	}

	license, err := kotsutil.LoadLicenseFromPath(filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to load license from path")
	}

	configValues, err := kotsutil.LoadConfigValuesFromFile(filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to load config values from path")
	}

	downstreams := []string{}
	for _, downstream := range a.Downstreams {
		downstreams = append(downstreams, downstream.Name)
	}

	k8sNamespace := "default"
	if os.Getenv("DEV_NAMESPACE") != "" {
		k8sNamespace = os.Getenv("DEV_NAMESPACE")
	}
	if os.Getenv("POD_NAMESPACE") != "" {
		k8sNamespace = os.Getenv("POD_NAMESPACE")
	}

	reOptions := rewrite.RewriteOptions{
		RootDir:          archiveDir,
		UpstreamURI:      fmt.Sprintf("replicated://%s", license.Spec.AppSlug),
		UpstreamPath:     filepath.Join(archiveDir, "upstream"),
		Installation:     installation,
		Downstreams:      downstreams,
		Silent:           true,
		CreateAppDir:     false,
		ExcludeKotsKinds: true,
		License:          license,
		ConfigValues:     configValues,
		K8sNamespace:     k8sNamespace,
		CopyImages:       false,
		IsAirgap:         a.IsAirgap,
	}

	if a.RegistrySettings != nil {
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

		reOptions.RegistryEndpoint = a.RegistrySettings.Hostname
		reOptions.RegistryNamespace = a.RegistrySettings.Namespace
		reOptions.RegistryUsername = a.RegistrySettings.Username
		reOptions.RegistryPassword = string(decryptedPassword)
	}

	err = rewrite.Rewrite(reOptions)
	if err != nil {
		return errors.Wrap(err, "rewrite directory")
	}
	return nil
}
