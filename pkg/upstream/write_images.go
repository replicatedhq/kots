package upstream

import (
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/logger"
)

type WriteUpstreamImageOptions struct {
	RootDir        string
	CreateAppDir   bool
	AppSlug        string
	SourceRegistry registry.RegistryOptions
	Log            *logger.Logger
}

func (u *Upstream) WriteUpstreamImages(options WriteUpstreamImageOptions) error {
	rootDir := options.RootDir
	if options.CreateAppDir {
		rootDir = path.Join(rootDir, u.Name)
	}
	upstreamDir := path.Join(rootDir, "upstream")
	imagesDir := path.Join(rootDir, "images")

	_, err := os.Stat(imagesDir)
	if err == nil {
		if err := os.RemoveAll(imagesDir); err != nil {
			return errors.Wrap(err, "failed to remove existing images")
		}
	}

	if err := image.SaveImages(options.SourceRegistry, options.AppSlug, options.Log, imagesDir, upstreamDir); err != nil {
		return errors.Wrap(err, "failed to save images")
	}

	return nil
}
