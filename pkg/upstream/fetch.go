package upstream

import (
	"net/url"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
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

	if fetchOptions.EncryptionKey != "" {
		err := crypto.InitFromString(fetchOptions.EncryptionKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create cipher")
		}
	}

	// change somehting
	u, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return nil, errors.Wrap(err, "parse request uri failed")
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
			pickVersionIsRequired(fetchOptions),
			pickReplicatedRegistryDomain(fetchOptions),
			pickReplicatedProxyDomain(fetchOptions),
			pickReplicatedChartNames(fetchOptions),
			fetchOptions.AppSlug,
			fetchOptions.AppSequence,
			fetchOptions.Airgap != nil,
			fetchOptions.Airgap,
			fetchOptions.LocalRegistry,
			fetchOptions.ReportingInfo,
			fetchOptions.SkipCompatibilityCheck,
		)
	}

	return nil, errors.Errorf("unknown protocol scheme %q", u.Scheme)
}

func pickReplicatedProxyDomain(fetchOptions *types.FetchOptions) string {
	if fetchOptions.Airgap != nil {
		return "" // custom domains are not applicable in airgap mode
	}
	return fetchOptions.CurrentReplicatedProxyDomain
}

func pickReplicatedRegistryDomain(fetchOptions *types.FetchOptions) string {
	if fetchOptions.Airgap != nil {
		return "" // custom domains are not applicable in airgap mode
	}
	return fetchOptions.CurrentReplicatedRegistryDomain
}

func pickVersionIsRequired(fetchOptions *types.FetchOptions) bool {
	if fetchOptions.Airgap != nil {
		return fetchOptions.Airgap.Spec.IsRequired
	}
	return fetchOptions.CurrentVersionIsRequired
}

func pickVersionLabel(fetchOptions *types.FetchOptions) string {
	if fetchOptions.Airgap != nil && fetchOptions.Airgap.Spec.VersionLabel != "" {
		return fetchOptions.Airgap.Spec.VersionLabel
	}

	// only initial install can request a specific version label
	if fetchOptions.AppSequence == 0 && fetchOptions.AppVersionLabel != "" {
		return fetchOptions.AppVersionLabel
	}

	return fetchOptions.CurrentVersionLabel
}

func pickCursor(fetchOptions *types.FetchOptions) replicatedapp.ReplicatedCursor {
	if fetchOptions.Airgap != nil && fetchOptions.Airgap.Spec.UpdateCursor != "" {
		return replicatedapp.ReplicatedCursor{
			ChannelID:   fetchOptions.Airgap.Spec.ChannelID,
			ChannelName: fetchOptions.Airgap.Spec.ChannelName,
			Cursor:      fetchOptions.Airgap.Spec.UpdateCursor,
		}
	}
	return replicatedapp.ReplicatedCursor{
		ChannelID:   fetchOptions.CurrentChannelID,
		ChannelName: fetchOptions.CurrentChannelName,
		Cursor:      fetchOptions.CurrentCursor,
	}
}

func pickReplicatedChartNames(fetchOptions *types.FetchOptions) []string {
	if fetchOptions.Airgap != nil {
		return fetchOptions.Airgap.Spec.ReplicatedChartNames
	}
	return fetchOptions.CurrentReplicatedChartNames
}
