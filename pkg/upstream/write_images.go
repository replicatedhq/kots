package upstream

import (
	"io"
	"path"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	kustomizeimage "sigs.k8s.io/kustomize/v3/pkg/image"
)

type WriteUpstreamImageOptions struct {
	RootDir        string
	CreateAppDir   bool
	AppSlug        string
	SourceRegistry registry.RegistryOptions
	DestRegistry   registry.RegistryOptions
	Log            *logger.Logger
	ReportWriter   io.Writer
}

func CopyUpstreamImages(u *types.Upstream, options WriteUpstreamImageOptions) ([]kustomizeimage.Image, error) {
	rootDir := options.RootDir
	if options.CreateAppDir {
		rootDir = path.Join(rootDir, u.Name)
	}
	upstreamDir := path.Join(rootDir, "upstream")

	newImages, err := image.CopyImages(options.SourceRegistry, options.DestRegistry, options.AppSlug, options.Log, options.ReportWriter, upstreamDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save images")
	}

	return newImages, nil
}
