package upstream

import (
	"net/url"

	"github.com/pkg/errors"
	types "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
)

func GetUpdatesUpstream(upstreamURI string, fetchOptions *types.FetchOptions) (*types.UpdateCheckResult, error) {
	if !util.IsURL(upstreamURI) {
		return nil, errors.New("not implemented")
	}

	u, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return nil, errors.Wrap(err, "parse request uri failed")
	}
	if u.Scheme == "replicated" {
		return getUpdatesReplicated(fetchOptions)
	}

	return nil, errors.Errorf("unknown protocol scheme %q", u.Scheme)
}
