package base

import (
	"io"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/logger"
	kustomizeimage "sigs.k8s.io/kustomize/api/types"
)

type WriteUpstreamImageOptions struct {
	BaseDir        string
	AppSlug        string
	SourceRegistry registry.RegistryOptions
	DestRegistry   registry.RegistryOptions
	CopyImages     bool
	IsAirgap       bool
	Log            *logger.CLILogger
	ReportWriter   io.Writer
	Installation   *kotsv1beta1.Installation
	Application    *kotsv1beta1.Application
}

type WriteUpstreamImageResult struct {
	Images        []kustomizeimage.Image          // images to be rewritten
	CheckedImages []kotsv1beta1.InstallationImage // all images found in the installation
}

func ProcessUpstreamImages(options WriteUpstreamImageOptions) (*WriteUpstreamImageResult, error) {
	additionalImages := make([]string, 0)
	if options.Application != nil {
		additionalImages = options.Application.Spec.AdditionalImages
	}
	checkedImages := makeImageInfoMap(options.Installation.Spec.KnownImages)

	rewriteAll := options.IsAirgap
	if options.Application != nil && options.Application.Spec.ProxyPublicImages {
		rewriteAll = true
	}

	newImages, err := image.ProcessImages(options.SourceRegistry, options.DestRegistry, options.AppSlug, options.Log, options.ReportWriter, options.BaseDir, additionalImages, options.CopyImages, rewriteAll, checkedImages)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save images")
	}

	return &WriteUpstreamImageResult{
		Images:        newImages,
		CheckedImages: makeInstallationImages(checkedImages),
	}, nil
}
