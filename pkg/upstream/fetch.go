package upstream

import (
	"net/url"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
)

func FetchUpstream(upstreamURI string, fetchOptions *types.FetchOptions) (*types.Upstream, error) {
	upstream, err := downloadUpstream(upstreamURI, fetchOptions)
	if err != nil {
		return nil, errors.Wrap(err, "download upstream failed")
	}

	return upstream, nil
}

func downloadUpstream(upstreamURI string, fetchOptions *types.FetchOptions) (*types.Upstream, error) {
	if !util.IsURL(upstreamURI) {
		return readFilesFromPath(upstreamURI)
	}

	var cipher *crypto.AESCipher
	if fetchOptions.EncryptionKey != "" {
		c, err := crypto.AESCipherFromString(fetchOptions.EncryptionKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create cipher")
		}
		cipher = c
	}

	u, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return nil, errors.Wrap(err, "parse request uri failed")
	}
	if u.Scheme == "helm" {
		return downloadHelm(u, fetchOptions.HelmRepoURI)
	}
	if u.Scheme == "replicated" {
		return downloadReplicated(
			u,
			fetchOptions.LocalPath,
			fetchOptions.RootDir,
			fetchOptions.UseAppDir,
			fetchOptions.License,
			fetchOptions.ConfigValues,
			fetchOptions.IdentityConfig,
			pickCursor(fetchOptions),
			pickVersionLabel(fetchOptions),
			cipher,
			fetchOptions.AppSlug,
			fetchOptions.AppSequence,
			fetchOptions.Airgap != nil,
			fetchOptions.LocalRegistry,
			fetchOptions.ReportingInfo,
		)
	}
	if u.Scheme == "git" {
		return downloadGit(upstreamURI)
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		return downloadHttp(upstreamURI)
	}

	return nil, errors.Errorf("unknown protocol scheme %q", u.Scheme)
}

func pickVersionLabel(fetchOptions *types.FetchOptions) string {
	if fetchOptions.Airgap != nil && fetchOptions.Airgap.Spec.VersionLabel != "" {
		return fetchOptions.Airgap.Spec.VersionLabel
	}
	return fetchOptions.CurrentVersionLabel
}

func pickCursor(fetchOptions *types.FetchOptions) ReplicatedCursor {
	if fetchOptions.Airgap != nil && fetchOptions.Airgap.Spec.UpdateCursor != "" {
		return ReplicatedCursor{
			ChannelID:   fetchOptions.Airgap.Spec.ChannelID,
			ChannelName: fetchOptions.Airgap.Spec.ChannelName,
			Cursor:      fetchOptions.Airgap.Spec.UpdateCursor,
		}
	}
	return ReplicatedCursor{
		ChannelID:   fetchOptions.CurrentChannelID,
		ChannelName: fetchOptions.CurrentChannelName,
		Cursor:      fetchOptions.CurrentCursor,
	}
}
