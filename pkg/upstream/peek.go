package upstream

import (
	"net/url"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
)

type Update struct {
	Cursor string `json:"cursor"`
}

func PeekUpstream(upstreamURI string, fetchOptions *FetchOptions) ([]Update, error) {
	versions, err := peekUpstream(upstreamURI, fetchOptions)
	if err != nil {
		return nil, errors.Wrap(err, "download upstream failed")
	}

	return versions, nil
}

func peekUpstream(upstreamURI string, fetchOptions *FetchOptions) ([]Update, error) {
	if !util.IsURL(upstreamURI) {
		return nil, errors.New("not implemented")
	}

	u, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return nil, errors.Wrap(err, "parse request uri failed")
	}
	if u.Scheme == "helm" {
		return peekHelm(u, fetchOptions.HelmRepoURI)
	}
	if u.Scheme == "replicated" {
		return peekReplicated(u, fetchOptions.LocalPath, fetchOptions.License, fetchOptions.CurrentCursor)
	}
	if u.Scheme == "git" {
		// return peekGit(upstreamURI)
		// TODO
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		// return peekHttp(upstreamURI)
		// TODO
	}

	return nil, errors.Errorf("unknown protocol scheme %q", u.Scheme)
}
