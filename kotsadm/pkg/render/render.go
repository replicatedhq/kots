package render

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	downstreamtypes "github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	registrytypes "github.com/replicatedhq/kots/kotsadm/pkg/registry/types"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/rewrite"
	"github.com/replicatedhq/kots/pkg/template"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
)

// RenderFile renders a single file
// this is useful for upstream/kotskinds files that are not rendered in the dir
func RenderFile(kotsKinds *kotsutil.KotsKinds, registrySettings *registrytypes.RegistrySettings, appSlug string, sequence int64, isAirgap bool, inputContent []byte) ([]byte, error) {
	fixedUpContent, err := kotsutil.FixUpYAML(inputContent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fix up yaml")
	}

	return RenderContent(kotsKinds, registrySettings, appSlug, sequence, isAirgap, fixedUpContent)
}

// RenderContent renders any string/content
// this is useful for rendering single values, like a status informer
func RenderContent(kotsKinds *kotsutil.KotsKinds, registrySettings *registrytypes.RegistrySettings, appSlug string, sequence int64, isAirgap bool, inputContent []byte) ([]byte, error) {
	builder, err := NewBuilder(kotsKinds, registrySettings, appSlug, sequence, isAirgap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create builder")
	}

	rendered, err := builder.RenderTemplate(string(inputContent), string(inputContent))
	if err != nil {
		return nil, errors.Wrap(err, "failed to render")
	}

	return []byte(rendered), nil
}

func NewBuilder(kotsKinds *kotsutil.KotsKinds, registrySettings *registrytypes.RegistrySettings, appSlug string, sequence int64, isAirgap bool) (*template.Builder, error) {
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
		return nil, errors.Wrap(err, "failed to load encryption cipher")
	}

	configGroups := []kotsv1beta1.ConfigGroup{}
	if kotsKinds.Config != nil && kotsKinds.Config.Spec.Groups != nil {
		configGroups = kotsKinds.Config.Spec.Groups
	}

	appInfo := template.ApplicationInfo{
		Slug: appSlug,
	}

	versionInfo := template.VersionInfoFromInstallation(sequence, isAirgap, kotsKinds.Installation.Spec)

	builderOptions := template.BuilderOptions{
		ConfigGroups:    configGroups,
		ExistingValues:  templateContextValues,
		LocalRegistry:   localRegistry,
		Cipher:          appCipher,
		License:         kotsKinds.License,
		ApplicationInfo: &appInfo,
		VersionInfo:     &versionInfo,
		IdentityConfig:  kotsKinds.IdentityConfig,
	}
	builder, _, err := template.NewBuilder(builderOptions)
	return &builder, errors.Wrap(err, "failed to create builder")
}

// RenderDir renders an app archive dir
// this is useful for when the license/config have updated, and template functions need to be evaluated again
func RenderDir(archiveDir string, a *apptypes.App, downstreams []downstreamtypes.Downstream, registrySettings *registrytypes.RegistrySettings, reportingInfo *upstreamtypes.ReportingInfo) error {
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

	downstreamNames := []string{}
	for _, d := range downstreams {
		downstreamNames = append(downstreamNames, d.Name)
	}

	appNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	reOptions := rewrite.RewriteOptions{
		RootDir:          archiveDir,
		UpstreamURI:      fmt.Sprintf("replicated://%s", license.Spec.AppSlug),
		UpstreamPath:     filepath.Join(archiveDir, "upstream"),
		Installation:     installation,
		Downstreams:      downstreamNames,
		Silent:           true,
		CreateAppDir:     false,
		ExcludeKotsKinds: true,
		License:          license,
		ConfigValues:     configValues,
		K8sNamespace:     appNamespace,
		CopyImages:       false,
		IsAirgap:         a.IsAirgap,
		AppSlug:          a.Slug,
		IsGitOps:         a.IsGitOps,
		AppSequence:      a.CurrentSequence + 1, // sequence +1 because this is the current latest sequence, not the sequence that the rendered version will be saved as
		ReportingInfo:    reportingInfo,
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

		reOptions.RegistryEndpoint = registrySettings.Hostname
		reOptions.RegistryNamespace = registrySettings.Namespace
		reOptions.RegistryUsername = registrySettings.Username
		reOptions.RegistryPassword = string(decryptedPassword)
	}

	err = rewrite.Rewrite(reOptions)
	if err != nil {
		return errors.Wrap(err, "rewrite directory")
	}
	return nil
}
