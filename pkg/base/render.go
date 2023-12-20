package base

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
)

type RenderOptions struct {
	SplitMultiDocYAML bool
	Namespace         string
	HelmVersion       string
	HelmValues        map[string]interface{}
	RegistrySettings  registrytypes.RegistrySettings
	ExcludeKotsKinds  bool
	AppSlug           string
	Sequence          int64
	IsAirgap          bool
	UseHelmInstall    bool
	Log               *logger.CLILogger
}

// RenderKotsKinds is responsible for rendering KOTS custom resources
func RenderKotsKinds(u *upstreamtypes.Upstream, renderOptions *RenderOptions) (map[string][]byte, error) {
	renderedKotsKinds, err := renderKotsKinds(u, renderOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render kots kinds")
	}

	return renderedKotsKinds, nil
}

// RenderUpstream is responsible for any conversions or transpilation steps are required
// to take an upstream and make it a valid kubernetes base
func RenderUpstream(u *upstreamtypes.Upstream, renderOptions *RenderOptions, renderedKotsKinds *kotsutil.KotsKinds) (base *Base, helmBases []Base, err error) {
	if u.Type == "helm" {
		base, err = RenderHelm(u, renderOptions)
		return
	}

	if u.Type == "replicated" {
		base, helmBases, err = renderReplicated(u, renderOptions, renderedKotsKinds)
		return
	}

	return nil, nil, errors.New("unknown upstream type")
}
