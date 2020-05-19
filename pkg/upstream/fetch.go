package upstream

import (
	"net/url"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
)

type FetchOptions struct {
	RootDir             string
	UseAppDir           bool
	HelmRepoName        string
	HelmRepoURI         string
	HelmOptions         []string
	LocalPath           string
	License             *kotsv1beta1.License
	UnsignedLicense     *kotsv1beta1.UnsignedLicense
	ConfigValues        *kotsv1beta1.ConfigValues
	Airgap              *kotsv1beta1.Airgap
	EncryptionKey       string
	CurrentCursor       string
	CurrentChannel      string
	CurrentVersionLabel string
}

func FetchUpstream(upstreamURI string, fetchOptions *FetchOptions) (*types.Upstream, error) {
	upstream, err := downloadUpstream(upstreamURI, fetchOptions)
	if err != nil {
		return nil, errors.Wrap(err, "download upstream failed")
	}

	return upstream, nil
}

func downloadUpstream(upstreamURI string, fetchOptions *FetchOptions) (*types.Upstream, error) {
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
		return downloadReplicated(u, fetchOptions.LocalPath, fetchOptions.RootDir, fetchOptions.UseAppDir, fetchOptions.License, fetchOptions.ConfigValues, pickCursor(fetchOptions), pickVersionLabel(fetchOptions), cipher)
	}
	if u.Scheme == "git" {
		return downloadGit(upstreamURI)
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		return downloadHttp(upstreamURI, fetchOptions, fetchOptions.ConfigValues, pickCursor(fetchOptions), cipher)
	}

	return nil, errors.Errorf("unknown protocol scheme %q", u.Scheme)
}

func pickVersionLabel(fetchOptions *FetchOptions) string {
	if fetchOptions.Airgap != nil && fetchOptions.Airgap.Spec.VersionLabel != "" {
		return fetchOptions.Airgap.Spec.VersionLabel
	}
	return fetchOptions.CurrentVersionLabel
}

func pickCursor(fetchOptions *FetchOptions) ReplicatedCursor {
	if fetchOptions.Airgap != nil && fetchOptions.Airgap.Spec.UpdateCursor != "" {
		return ReplicatedCursor{
			ChannelName: fetchOptions.Airgap.Spec.ChannelName,
			Cursor:      fetchOptions.Airgap.Spec.UpdateCursor,
		}
	}
	return ReplicatedCursor{
		ChannelName: fetchOptions.CurrentChannel,
		Cursor:      fetchOptions.CurrentCursor,
	}
}
