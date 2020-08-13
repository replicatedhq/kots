package base

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
)

type RenderOptions struct {
	SplitMultiDocYAML      bool
	Namespace              string
	HelmVersion            string
	HelmOptions            []string
	LocalRegistryHost      string
	LocalRegistryNamespace string
	LocalRegistryUsername  string
	LocalRegistryPassword  string
	ExcludeKotsKinds       bool
	ExtractKotsHookEvents  bool
	Log                    *logger.Logger
}

// RenderUpstream is responsible for any conversions or transpilation steps are required
// to take an upstream and make it a valid kubernetes base
func RenderUpstream(u *upstreamtypes.Upstream, renderOptions *RenderOptions) (*Base, map[HookEvent]*Base, error) {
	if u.Type == "helm" {
		base, err := RenderHelm(u, renderOptions)
		return base, nil, err
	}

	if u.Type == "replicated" {
		return renderReplicated(u, renderOptions)
	}

	return nil, nil, errors.New("unknown upstream type")
}
