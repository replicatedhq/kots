package base

import (
	"io"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/logger"
	kustomizeimage "sigs.k8s.io/kustomize/v3/pkg/image"
)

type WriteUpstreamImageOptions struct {
	BaseDir        string
	AppSlug        string
	SourceRegistry registry.RegistryOptions
	DestRegistry   registry.RegistryOptions
	Log            *logger.Logger
	ReportWriter   io.Writer
}

func CopyUpstreamImages(options WriteUpstreamImageOptions) ([]kustomizeimage.Image, error) {
	newImages, err := image.CopyImages(options.SourceRegistry, options.DestRegistry, options.AppSlug, options.Log, options.ReportWriter, options.BaseDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save images")
	}

	return newImages, nil
}
