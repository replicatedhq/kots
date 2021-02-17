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
	HelmValues             map[string]interface{}
	LocalRegistryHost      string
	LocalRegistryNamespace string
	LocalRegistryUsername  string
	LocalRegistryPassword  string
	ExcludeKotsKinds       bool
	AppSlug                string
	Sequence               int64
	IsAirgap               bool
	Log                    *logger.CLILogger
}

// RenderUpstream is responsible for any conversions or transpilation steps are required
// to take an upstream and make it a valid kubernetes base
func RenderUpstream(u *upstreamtypes.Upstream, renderOptions *RenderOptions) (*Base, error) {
	if u.Type == "helm" {
		return RenderHelm(u, renderOptions)
	}

	if u.Type == "replicated" {
		return renderReplicated(u, renderOptions)
	}

	return nil, errors.New("unknown upstream type")
}
