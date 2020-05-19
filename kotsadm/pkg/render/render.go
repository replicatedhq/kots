package render

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	registrytypes "github.com/replicatedhq/kots/kotsadm/pkg/registry/types"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/rewrite"
	"github.com/replicatedhq/kots/pkg/template"
)

// RenderFile renders a single file
// this is useful for upstream/kotskinds files that are not rendered in the dir
func RenderFile(kotsKinds *kotsutil.KotsKinds, registrySettings *registrytypes.RegistrySettings, inputContent []byte) ([]byte, error) {
	inputContent, err := kotsutil.FixUpYAML(inputContent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fix up yaml")
	}

	apiCipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load apiCipher")
	}

	localRegistry := template.LocalRegistry{}

	if registrySettings != nil {
		decodedPassword, err := base64.StdEncoding.DecodeString(registrySettings.PasswordEnc)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode")
		}

		decryptedPassword, err := apiCipher.Decrypt([]byte(decodedPassword))
		if err != nil {
			return nil, errors.Wrap(err, "failed to decrypt")
		}

		localRegistry.Host = registrySettings.Hostname
		localRegistry.Namespace = registrySettings.Namespace
		localRegistry.Username = registrySettings.Username
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

	appCipher, err := crypto.AESCipherFromString(kotsKinds.Installation.Spec.EncryptionKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load appCipher")
	}

	configGroups := []kotsv1beta1.ConfigGroup{}
	if kotsKinds.Config != nil && kotsKinds.Config.Spec.Groups != nil {
		configGroups = kotsKinds.Config.Spec.Groups
	}

	builder, _, err := template.NewBuilder(configGroups, templateContextValues, localRegistry, appCipher, kotsKinds.License, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create builder")
	}

	rendered, err := builder.RenderTemplate(string(inputContent), string(inputContent))
	if err != nil {
		return nil, errors.Wrap(err, "failed to render")
	}

	return []byte(rendered), nil
}

// RenderDir renders an app archive dir
// this is useful for when the license/config have updated, and template functions need to be evaluated again
func RenderDir(archiveDir string, appID string, appSequence int64, registrySettings *registrytypes.RegistrySettings) error {
	installation, err := kotsutil.LoadInstallationFromPath(filepath.Join(archiveDir, "upstream", "userdata", "installation.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to load installation from path")
	}

	license, unsignedLicense, err := kotsutil.LoadLicenseFromPath(filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to load license from path")
	}

	configValues, err := kotsutil.LoadConfigValuesFromFile(filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"))
	if err != nil && !os.IsNotExist(errors.Cause(err)) {
		return errors.Wrap(err, "failed to load config values from path")
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

	appNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	a, err := app.Get(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	upstreamURI := ""
	if license != nil {
		upstreamURI = fmt.Sprintf("replicated://%s", license.Spec.AppSlug)
	} else if unsignedLicense != nil {
		upstreamURI = unsignedLicense.Spec.Endpoint
	}
	rewriteOptions := rewrite.RewriteOptions{
		RootDir:          archiveDir,
		UpstreamURI:      upstreamURI,
		UpstreamPath:     filepath.Join(archiveDir, "upstream"),
		Installation:     installation,
		Downstreams:      downstreamNames,
		Silent:           true,
		CreateAppDir:     false,
		ExcludeKotsKinds: true,
		License:          license,
		UnsignedLicense:  unsignedLicense,
		ConfigValues:     configValues,
		K8sNamespace:     appNamespace,
		CopyImages:       false,
		IsAirgap:         a.IsAirgap,
		AppSlug:          a.Slug,
		AppSequence:      appSequence,
		IsGitOps:         a.IsGitOps,
	}

	if registrySettings != nil {
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

		rewriteOptions.RegistryEndpoint = registrySettings.Hostname
		rewriteOptions.RegistryNamespace = registrySettings.Namespace
		rewriteOptions.RegistryUsername = registrySettings.Username
		rewriteOptions.RegistryPassword = string(decryptedPassword)
	}

	err = rewrite.Rewrite(rewriteOptions)
	if err != nil {
		return errors.Wrap(err, "rewrite directory")
	}
	return nil
}
