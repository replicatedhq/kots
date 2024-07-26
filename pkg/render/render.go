package render

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	types "github.com/replicatedhq/kots/pkg/render/types"
	"github.com/replicatedhq/kots/pkg/rewrite"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

type Renderer struct {
}

// RenderFile renders a single file
// this is useful for upstream/kotskinds files that are not rendered in the dir
func (r Renderer) RenderFile(opts types.RenderFileOptions) ([]byte, error) {
	return RenderFile(opts)
}

func RenderFile(opts types.RenderFileOptions) ([]byte, error) {
	fixedUpContent, err := kotsutil.FixUpYAML(opts.InputContent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fix up yaml")
	}
	opts.InputContent = fixedUpContent

	return RenderContent(opts)
}

// RenderContent renders any string/content
// this is useful for rendering single values, like a status informer
func RenderContent(opts types.RenderFileOptions) ([]byte, error) {
	builder, err := NewBuilder(opts.KotsKinds, opts.RegistrySettings, opts.AppSlug, opts.Sequence, opts.IsAirgap, opts.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create builder")
	}

	rendered, err := builder.RenderTemplate(string(opts.InputContent), string(opts.InputContent))
	if err != nil {
		return nil, errors.Wrap(err, "failed to render")
	}

	return []byte(rendered), nil
}

func NewBuilder(kotsKinds *kotsutil.KotsKinds, registrySettings registrytypes.RegistrySettings, appSlug string, sequence int64, isAirgap bool, namespace string) (*template.Builder, error) {
	templateContextValues := make(map[string]template.ItemValue)
	if kotsKinds.ConfigValues != nil {
		for k, v := range kotsKinds.ConfigValues.Spec.Values {
			templateContextValues[k] = template.ItemValue{
				Value:   v.Value,
				Default: v.Default,
			}
		}
	}

	err := crypto.InitFromString(kotsKinds.Installation.Spec.EncryptionKey)
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

	versionInfo := template.VersionInfoFromInstallationSpec(sequence, isAirgap, kotsKinds.Installation.Spec)

	builderOptions := template.BuilderOptions{
		ConfigGroups:    configGroups,
		ExistingValues:  templateContextValues,
		LocalRegistry:   registrySettings,
		License:         kotsKinds.License,
		Application:     &kotsKinds.KotsApplication,
		ApplicationInfo: &appInfo,
		VersionInfo:     &versionInfo,
		IdentityConfig:  kotsKinds.IdentityConfig,
		Namespace:       namespace,
		DecryptValues:   true,
	}
	builder, _, err := template.NewBuilder(builderOptions)
	return &builder, errors.Wrap(err, "failed to create builder")
}

// RenderDir renders an app archive dir
// this is useful for when the license/config have updated, and template functions need to be evaluated again
func (r Renderer) RenderDir(opts types.RenderDirOptions) error {
	return RenderDir(opts)
}

func RenderDir(opts types.RenderDirOptions) error {
	installation, err := kotsutil.LoadInstallationFromPath(filepath.Join(opts.ArchiveDir, "upstream", "userdata", "installation.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to load installation from path")
	}

	license, err := kotsutil.LoadLicenseFromPath(filepath.Join(opts.ArchiveDir, "upstream", "userdata", "license.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to load license from path")
	}

	configValues, err := kotsutil.LoadConfigValuesFromFile(filepath.Join(opts.ArchiveDir, "upstream", "userdata", "config.yaml"))
	if err != nil && !os.IsNotExist(errors.Cause(err)) {
		return errors.Wrap(err, "failed to load config values from path")
	}

	downstreamNames := []string{}
	for _, d := range opts.Downstreams {
		downstreamNames = append(downstreamNames, d.Name)
	}

	appNamespace := util.PodNamespace
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}
	reOptions := rewrite.RewriteOptions{
		RootDir:          opts.ArchiveDir,
		UpstreamURI:      fmt.Sprintf("replicated://%s", license.Spec.AppSlug),
		UpstreamPath:     filepath.Join(opts.ArchiveDir, "upstream"),
		Installation:     installation,
		Downstreams:      downstreamNames,
		Silent:           true,
		CreateAppDir:     false,
		ExcludeKotsKinds: true,
		License:          license,
		ConfigValues:     configValues,
		K8sNamespace:     appNamespace,
		CopyImages:       false,
		IsAirgap:         opts.App.IsAirgap,
		AppID:            opts.App.ID,
		AppSlug:          opts.App.Slug,
		AppChannelID:     opts.App.ChannelID,
		IsGitOps:         opts.App.IsGitOps,
		AppSequence:      opts.Sequence,
		ReportingInfo:    opts.ReportingInfo,
		RegistrySettings: opts.RegistrySettings,

		// TODO: pass in as arguments if this is ever called from CLI
		HTTPProxyEnvValue:  os.Getenv("HTTP_PROXY"),
		HTTPSProxyEnvValue: os.Getenv("HTTPS_PROXY"),
		NoProxyEnvValue:    os.Getenv("NO_PROXY"),
	}

	err = rewrite.Rewrite(reOptions)
	if err != nil {
		return errors.Wrap(err, "rewrite directory")
	}
	return nil
}
