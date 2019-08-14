package pull

import (
	"path"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/downstream"
	"github.com/replicatedhq/kots/pkg/logger"
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
	log := logger.NewLogger()
	log.Info("")

	fetchOptions := upstream.FetchOptions{}
	fetchOptions.HelmRepoURI = pullOptions.HelmRepoURI

	log.Info("Pulling upstream")
	u, err := upstream.FetchUpstream(upstreamURI, &fetchOptions)
	if err != nil {
		return errors.Wrap(err, "failed to fetch upstream")
	}

	writeUpstreamOptions := upstream.WriteOptions{
		RootDir:      pullOptions.RootDir,
		CreateAppDir: true,
		Overwrite:    pullOptions.Overwrite,
	}
	log.Info("Writing upstream")
	if err := u.WriteUpstream(writeUpstreamOptions); err != nil {
		return errors.Wrap(err, "failed to write upstream")
	}

	renderOptions := base.RenderOptions{
		SplitMultiDocYAML: true,
		Namespace:         pullOptions.Namespace,
	}
	log.Info("Creating base")
	b, err := base.RenderUpstream(u, &renderOptions)
	if err != nil {
		return errors.Wrap(err, "failed to render upstream")
	}

	writeBaseOptions := base.WriteOptions{
		BaseDir:   u.GetBaseDir(writeUpstreamOptions),
		Overwrite: pullOptions.Overwrite,
	}
	log.Info("Writing base")
	if err := b.WriteBase(writeBaseOptions); err != nil {
		return errors.Wrap(err, "failed to write base")
	}

	log.Info("Creating midstream")
	m, err := midstream.CreateMidstream(b)
	if err != nil {
		return errors.Wrap(err, "failed to create midstream")
	}

	writeMidstreamOptions := midstream.WriteOptions{
		MidstreamDir: path.Join(b.GetOverlaysDir(writeBaseOptions), "midstream"),
		BaseDir:      u.GetBaseDir(writeUpstreamOptions),
		Overwrite:    pullOptions.Overwrite,
	}
	log.Info("Writing midstream")
	if err := m.WriteMidstream(writeMidstreamOptions); err != nil {
		return errors.Wrap(err, "failed to write midstream")
	}

	for _, downstreamName := range pullOptions.Downstreams {
		log.Info("Creating downstream %q", downstreamName)
		d, err := downstream.CreateDownstream(m, downstreamName)
		if err != nil {
			return errors.Wrap(err, "failed to create downstream")
		}

		writeDownstreamOptions := downstream.WriteOptions{
			DownstreamDir: path.Join(b.GetOverlaysDir(writeBaseOptions), "downstreams", downstreamName),
			MidstreamDir:  writeMidstreamOptions.MidstreamDir,
			Overwrite:     pullOptions.Overwrite,
		}

		log.Info("writing downstream %q to %s", downstreamName, writeDownstreamOptions.DownstreamDir)
		if err := d.WriteDownstream(writeDownstreamOptions); err != nil {
			return errors.Wrap(err, "failed to write downstream")
		}
	}
	return nil
}
