package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/rewrite"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	yaml "github.com/replicatedhq/yaml/v3"
)

// RenderFile renders a single file
// this is useful for upstream/kotskinds files that are not rendered in the dir
func (a *App) RenderFile(kotsKinds *kotsutil.KotsKinds, inputContent []byte) ([]byte, error) {

	yamlObj := map[string]interface{}{}
	err := yaml.Unmarshal(inputContent, &yamlObj)
	if err != nil {
		return nil, err
	}
	inputContent, err = util.MarshalIndent(2, yamlObj)
	if err != nil {
		return nil, err
	}

	apiCipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load apiCipher")
	}

	localRegistry := template.LocalRegistry{}

	if a.RegistrySettings != nil {
		decodedPassword, err := base64.StdEncoding.DecodeString(a.RegistrySettings.PasswordEnc)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode")
		}

		decryptedPassword, err := apiCipher.Decrypt([]byte(decodedPassword))
		if err != nil {
			return nil, errors.Wrap(err, "failed to decrypt")
		}

		localRegistry.Host = a.RegistrySettings.Hostname
		localRegistry.Namespace = a.RegistrySettings.Namespace
		localRegistry.Username = a.RegistrySettings.Username
		localRegistry.Password = string(decryptedPassword)
	}

	templateContextValues := make(map[string]template.ItemValue)
	if kotsKinds.ConfigValues != nil {
		for k, v := range kotsKinds.ConfigValues.Spec.Values {
			templateContextValues[k] = template.ItemValue{
				Value:   v.Value,
				Default: v.Default,
			}
		}
	}

	builder := template.Builder{
		Ctx: []template.Ctx{
			template.LicenseCtx{License: kotsKinds.License},
			template.StaticCtx{},
		},
	}

	appCipher, err := crypto.AESCipherFromString(kotsKinds.Installation.Spec.EncryptionKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load appCipher")
	}

	configGroups := []kotsv1beta1.ConfigGroup{}
	if kotsKinds.Config != nil && kotsKinds.Config.Spec.Groups != nil {
		configGroups = kotsKinds.Config.Spec.Groups
	}

	configCtx, err := builder.NewConfigContext(configGroups, templateContextValues, localRegistry, appCipher, kotsKinds.License)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create builder")
	}
	builder.AddCtx(configCtx)

	rendered, err := builder.RenderTemplate(string(inputContent), string(inputContent))
	if err != nil {
		return nil, errors.Wrap(err, "failed to render")
	}

	return []byte(rendered), nil
}

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
	if err != nil && !os.IsNotExist(errors.Cause(err)) {
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
