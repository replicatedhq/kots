package specs

import (
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	version "github.com/hashicorp/go-version"
	"github.com/replicatedhq/ship/pkg/util"
)

const (
	UpstreamVersionToken = "_latest_"
)

func (r *Resolver) maybeCreateVersionedUpstream(upstream string) (string, error) {
	debug := log.With(level.Debug(r.Logger), "method", "maybeCreateVersionedUpstream")
	if util.IsGithubURL(upstream) {
		githubURL, err := util.ParseGithubURL(upstream, "master")
		if err != nil {
			debug.Log("event", "parseGithubURL.fail")
			return upstream, nil
		}

		isSemver := len(strings.Split(githubURL.Ref, ".")) > 1
		parsedVersion, err := version.NewVersion(githubURL.Ref)
		if err == nil && isSemver {
			return strings.Replace(upstream, parsedVersion.Original(), UpstreamVersionToken, 1), nil
		}
	}

	return upstream, nil
}
