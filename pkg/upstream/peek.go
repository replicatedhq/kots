package upstream

import (
	"net/url"

	"github.com/pkg/errors"
	types "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
)

type Update struct {
	Cursor       string `json:"cursor"`
	VersionLabel string `json:"versionLabel"`
}

func GetUpdatesUpstream(upstreamURI string, fetchOptions *types.FetchOptions) ([]Update, error) {
	versions, err := getUpdatesUpstream(upstreamURI, fetchOptions)
	if err != nil {
		return nil, errors.Wrap(err, "download upstream failed")
	}

	return versions, nil
}

func getUpdatesUpstream(upstreamURI string, fetchOptions *types.FetchOptions) ([]Update, error) {
	if !util.IsURL(upstreamURI) {
		return nil, errors.New("not implemented")
	}

	u, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return nil, errors.Wrap(err, "parse request uri failed")
	}
	if u.Scheme == "helm" {
		return getUpdatesHelm(u, fetchOptions.HelmRepoURI)
	}
	if u.Scheme == "replicated" {
		currentCursor := ReplicatedCursor{
			ChannelID:   fetchOptions.CurrentChannelID,
			ChannelName: fetchOptions.CurrentChannelName,
			Cursor:      fetchOptions.CurrentCursor,
		}
		return getUpdatesReplicated(u, fetchOptions.LocalPath, currentCursor, fetchOptions.CurrentVersionLabel, fetchOptions.License, fetchOptions.Reporting)
	}
	if u.Scheme == "git" {
		// return getUpdatesGit(upstreamURI)
		// TODO
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		// return getUpdatesHttp(upstreamURI)
		// TODO
	}

	return nil, errors.Errorf("unknown protocol scheme %q", u.Scheme)
}
