package base

import (
	"github.com/pkg/errors"
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

// RenderUpstream is responsible for any conversions or transpilation steps are required
// to take an upstream and make it a valid kubernetes base
func RenderUpstream(u *upstreamtypes.Upstream, renderOptions *RenderOptions) (base *Base, helmBases []Base, renderedKotsKinds map[string][]byte, err error) {
	if u.Type == "helm" {
		base, err = RenderHelm(u, renderOptions)
		return
	}

	if u.Type == "replicated" {
		base, helmBases, renderedKotsKinds, err = renderReplicated(u, renderOptions)
		return
	}

	return nil, nil, nil, errors.New("unknown upstream type")
}
