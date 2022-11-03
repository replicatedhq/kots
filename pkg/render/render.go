package render

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	kotsutiltypes "github.com/replicatedhq/kots/pkg/kotsutil/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/rewrite"
	"github.com/replicatedhq/kots/pkg/util"
)

type Renderer struct {
}

// RenderDir renders an app archive dir
// this is useful for when the license/config have updated, and template functions need to be evaluated again
func (r Renderer) RenderDir(archiveDir string, kotsKinds *kotsutiltypes.KotsKinds, a *apptypes.App, downstreams []downstreamtypes.Downstream, registrySettings registrytypes.RegistrySettings, sequence int64) error {
	return RenderDir(archiveDir, kotsKinds, a, downstreams, registrySettings, sequence)
}

func RenderDir(archiveDir string, kotsKinds *kotsutiltypes.KotsKinds, a *apptypes.App, downstreams []downstreamtypes.Downstream, registrySettings registrytypes.RegistrySettings, sequence int64) error {
	downstreamNames := []string{}
	for _, d := range downstreams {
		downstreamNames = append(downstreamNames, d.Name)
	}

	appNamespace := util.PodNamespace
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	reOptions := rewrite.RewriteOptions{
		RootDir:            archiveDir,
		UpstreamURI:        fmt.Sprintf("replicated://%s", kotsKinds.License.Spec.AppSlug),
		UpstreamPath:       filepath.Join(archiveDir, "upstream"),
		Downstreams:        downstreamNames,
		Silent:             true,
		CreateAppDir:       false,
		Installation:       &kotsKinds.Installation,
		License:            kotsKinds.License,
		ConfigValues:       kotsKinds.ConfigValues,
		IdentityConfig:     kotsKinds.IdentityConfig,
		K8sNamespace:       appNamespace,
		CopyImages:         false,
		IsAirgap:           a.IsAirgap,
		AppID:              a.ID,
		AppSlug:            a.Slug,
		IsGitOps:           a.IsGitOps,
		AppSequence:        sequence,
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

	if err := rewrite.Rewrite(reOptions); err != nil {
		return errors.Wrap(err, "rewrite directory")
	}
	return nil
}
