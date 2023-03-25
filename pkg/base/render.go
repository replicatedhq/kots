package base

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
)

type RenderOptions struct {
	SplitMultiDocYAML       bool
	Namespace               string
	HelmVersion             string
	HelmOptions             []string
	HelmValues              map[string]interface{}
	LocalRegistryHost       string
	LocalRegistryNamespace  string
	LocalRegistryUsername   string
	LocalRegistryPassword   string
	LocalRegistryIsReadOnly bool
	ExcludeKotsKinds        bool
	AppSlug                 string
	Sequence                int64
	IsAirgap                bool
	UseHelmInstall          bool
	Log                     *logger.CLILogger
}

// RenderUpstream is responsible for any conversions or transpilation steps are required
// to take an upstream and make it a valid kubernetes base
func RenderUpstream(u *upstreamtypes.Upstream, renderOptions *RenderOptions) (base *Base, helmBases []Base, kotsBase *Base, err error) {
	if u.Type == "helm" {
		// TODO: create the kotsBase in this function if applicable
		base, err = RenderHelm(u, renderOptions)
		return
	}

	if u.Type == "replicated" {
		base, helmBases, kotsBase, err = renderReplicated(u, renderOptions)
		return
	}

	return nil, nil, nil, errors.New("unknown upstream type")
}
