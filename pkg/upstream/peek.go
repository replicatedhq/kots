package upstream

import (
	"net/url"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
)

type Update struct {
	Cursor       string `json:"cursor"`
	VersionLabel string `json:"versionLabel"`
}

func GetUpdatesUpstream(upstreamURI string, fetchOptions *FetchOptions) ([]Update, error) {
	versions, err := getUpdatesUpstream(upstreamURI, fetchOptions)
	if err != nil {
		return nil, errors.Wrap(err, "download upstream failed")
	}

	return versions, nil
}

func getUpdatesUpstream(upstreamURI string, fetchOptions *FetchOptions) ([]Update, error) {
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
		cursor := ReplicatedCursor{
			ChannelName: fetchOptions.CurrentChannel,
			Cursor:      fetchOptions.CurrentCursor,
		}
		return getUpdatesReplicated(u, fetchOptions.LocalPath, cursor, fetchOptions.CurrentVersionLabel, fetchOptions.License)
	}
	if u.Scheme == "git" {
		// return getUpdatesGit(upstreamURI)
		// TODO
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		cursor := ReplicatedCursor{
			ChannelName: fetchOptions.CurrentChannel,
			Cursor:      fetchOptions.CurrentCursor,
		}
		return getUpdatesHttp(u, fetchOptions.LocalPath, cursor, fetchOptions.CurrentVersionLabel, fetchOptions.UnsignedLicense)
	}

	return nil, errors.Errorf("unknown protocol scheme %q", u.Scheme)
}
