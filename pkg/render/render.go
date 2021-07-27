package render

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/rewrite"
	"github.com/replicatedhq/kots/pkg/template"
)

type Renderer struct {
}

// RenderFile renders a single file
// this is useful for upstream/kotskinds files that are not rendered in the dir
func (r Renderer) RenderFile(kotsKinds *kotsutil.KotsKinds, registrySettings registrytypes.RegistrySettings, appSlug string, sequence int64, isAirgap bool, namespace string, inputContent []byte) ([]byte, error) {
	return RenderFile(kotsKinds, registrySettings, appSlug, sequence, isAirgap, namespace, inputContent)
}

func RenderFile(kotsKinds *kotsutil.KotsKinds, registrySettings registrytypes.RegistrySettings, appSlug string, sequence int64, isAirgap bool, namespace string, inputContent []byte) ([]byte, error) {
	fixedUpContent, err := kotsutil.FixUpYAML(inputContent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fix up yaml")
	}

	return RenderContent(kotsKinds, registrySettings, appSlug, sequence, isAirgap, namespace, fixedUpContent)
}

// RenderContent renders any string/content
// this is useful for rendering single values, like a status informer
func RenderContent(kotsKinds *kotsutil.KotsKinds, registrySettings registrytypes.RegistrySettings, appSlug string, sequence int64, isAirgap bool, namespace string, inputContent []byte) ([]byte, error) {
	builder, err := NewBuilder(kotsKinds, registrySettings, appSlug, sequence, isAirgap, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create builder")
	}

	rendered, err := builder.RenderTemplate(string(inputContent), string(inputContent))
	if err != nil {
		return nil, errors.Wrap(err, "failed to render")
	}

	return []byte(rendered), nil
}

func NewBuilder(kotsKinds *kotsutil.KotsKinds, registrySettings registrytypes.RegistrySettings, appSlug string, sequence int64, isAirgap bool, namespace string) (*template.Builder, error) {
	localRegistry := template.LocalRegistry{
		Host:      registrySettings.Hostname,
		Namespace: registrySettings.Namespace,
		Username:  registrySettings.Username,
		Password:  registrySettings.Password,
		ReadOnly:  registrySettings.IsReadOnly,
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
		Namespace:       namespace,
	}
	builder, _, err := template.NewBuilder(builderOptions)
	return &builder, errors.Wrap(err, "failed to create builder")
}

// RenderDir renders an app archive dir
// this is useful for when the license/config have updated, and template functions need to be evaluated again
func (r Renderer) RenderDir(archiveDir string, a *apptypes.App, downstreams []downstreamtypes.Downstream, registrySettings registrytypes.RegistrySettings, createNewVersion bool) error {
	return RenderDir(archiveDir, a, downstreams, registrySettings, createNewVersion)
}

func RenderDir(archiveDir string, a *apptypes.App, downstreams []downstreamtypes.Downstream, registrySettings registrytypes.RegistrySettings, createNewVersion bool) error {
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
		RootDir:            archiveDir,
		UpstreamURI:        fmt.Sprintf("replicated://%s", license.Spec.AppSlug),
		UpstreamPath:       filepath.Join(archiveDir, "upstream"),
		Installation:       installation,
		Downstreams:        downstreamNames,
		Silent:             true,
		CreateAppDir:       false,
		ExcludeKotsKinds:   true,
		License:            license,
		ConfigValues:       configValues,
		K8sNamespace:       appNamespace,
		CopyImages:         false,
		IsAirgap:           a.IsAirgap,
		AppID:              a.ID,
		AppSlug:            a.Slug,
		IsGitOps:           a.IsGitOps,
		ReportingInfo:      reporting.GetReportingInfo(a.ID),
		RegistryEndpoint:   registrySettings.Hostname,
		RegistryNamespace:  registrySettings.Namespace,
		RegistryUsername:   registrySettings.Username,
		RegistryPassword:   registrySettings.Password,
		RegistryIsReadOnly: registrySettings.IsReadOnly,

		// TODO: pass in as arguments if this is ever called from CLI
		HTTPProxyEnvValue:  os.Getenv("HTTP_PROXY"),
		HTTPSProxyEnvValue: os.Getenv("HTTPS_PROXY"),
		NoProxyEnvValue:    os.Getenv("NO_PROXY"),
	}

	if createNewVersion {
		reOptions.AppSequence = a.CurrentSequence + 1
	} else {
		reOptions.AppSequence = a.CurrentSequence
	}

	err = rewrite.Rewrite(reOptions)
	if err != nil {
		return errors.Wrap(err, "rewrite directory")
	}
	return nil
}
