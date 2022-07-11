package base

import (
	"io"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	kustomizeimage "sigs.k8s.io/kustomize/api/types"
)

type WriteUpstreamImageOptions struct {
	BaseDir           string
	AppSlug           string
	SourceRegistry    registry.RegistryOptions
	DestRegistry      registry.RegistryOptions
	DockerHubRegistry registry.RegistryOptions
	CopyImages        bool
	IsAirgap          bool
	Log               *logger.CLILogger
	ReportWriter      io.Writer
	KotsKinds         *kotsutil.KotsKinds
}

type WriteUpstreamImageResult struct {
	Images        []kustomizeimage.Image          // images to be rewritten
	CheckedImages []kotsv1beta1.InstallationImage // all images found in the installation
}

func ProcessUpstreamImages(options WriteUpstreamImageOptions) (*WriteUpstreamImageResult, error) {
	rewriteAll := options.IsAirgap
	additionalImages := make([]string, 0)
	checkedImages := make(map[string]image.ImageInfo)

	if options.KotsKinds != nil {
		additionalImages = kotsutil.GetImagesFromKotsKinds(options.KotsKinds)
		checkedImages = makeImageInfoMap(options.KotsKinds.Installation.Spec.KnownImages)
		if options.KotsKinds.KotsApplication.Spec.ProxyPublicImages {
			rewriteAll = true
		}
	}

	newImages, err := image.ProcessImages(options.SourceRegistry, options.DestRegistry, options.AppSlug, options.Log, options.ReportWriter, options.BaseDir, additionalImages, options.CopyImages, rewriteAll, checkedImages, options.DockerHubRegistry)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save images")
	}

	return &WriteUpstreamImageResult{
		Images:        newImages,
		CheckedImages: makeInstallationImages(checkedImages),
	}, nil
}
