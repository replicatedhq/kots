package base

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/upstream"
)

type RenderOptions struct {
	SplitMultiDocYAML bool
	Namespace         string
}

// RenderUpstream is responsible for any conversions or transpilation steps are required
// to take an upstream and make it a valid kubernetes base
func RenderUpstream(u *upstream.Upstream, renderOptions *RenderOptions) (*Base, error) {
	if u.Type == "helm" {
		return renderHelm(u, renderOptions)
	}

	if u.Type == "replicated" {
		return renderReplicated(u, renderOptions)
	}

	return nil, errors.New("unknown upstream type")
}
