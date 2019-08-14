package pull

import (
	"path"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/upstream"
)

type PullOptions struct {
	HelmRepoURI string
	RootDir     string
	Overwrite   bool
	Namespace   string
	Downstreams []string
}

func Pull(upstreamURI string, pullOptions PullOptions) error {
	fetchOptions := upstream.FetchOptions{}
	fetchOptions.HelmRepoURI = pullOptions.HelmRepoURI

	u, err := upstream.FetchUpstream(upstreamURI, &fetchOptions)
	if err != nil {
		return errors.Wrap(err, "failed to fetch upstream")
	}

	writeUpstreamOptions := upstream.WriteOptions{
		RootDir:      pullOptions.RootDir,
		CreateAppDir: true,
		Overwrite:    pullOptions.Overwrite,
	}
	if err := u.WriteUpstream(writeUpstreamOptions); err != nil {
		return errors.Wrap(err, "failed to write upstream")
	}

	renderOptions := base.RenderOptions{
		SplitMultiDocYAML: true,
		Namespace:         pullOptions.Namespace,
	}
	b, err := base.RenderUpstream(u, &renderOptions)
	if err != nil {
		return errors.Wrap(err, "failed to render upstream")
	}

	writeBaseOptions := base.WriteOptions{
		BaseDir:   u.GetBaseDir(writeUpstreamOptions),
		Overwrite: pullOptions.Overwrite,
	}
	if err := b.WriteBase(writeBaseOptions); err != nil {
		return errors.Wrap(err, "failed to write base")
	}

	m, err := midstream.CreateMidstream(b)
	if err != nil {
		return errors.Wrap(err, "failed to create midstream")
	}

	writeMidstreamOptions := midstream.WriteOptions{
		MidstreamDir: path.Join(b.GetOverlaysDir(writeBaseOptions), "midstream"),
		BaseDir:      u.GetBaseDir(writeUpstreamOptions),
		Overwrite:    pullOptions.Overwrite,
	}
	if err := m.WriteMidstream(writeMidstreamOptions); err != nil {
		return errors.Wrap(err, "failed to write midstream")
	}

	// for _, downstream := range pullOptions.Downstreams {

	// }
	return nil
}
