package base

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/upstream"
)

type RenderOptions struct {
	SplitMultiDocYAML bool
	Namespace         string
}

func RenderUpstream(u *upstream.Upstream, renderOptions *RenderOptions) (*Base, error) {
	if u.Type == "helm" {
		return renderHelm(u, renderOptions)
	}

	return nil, errors.New("unknown upstream type")
}
