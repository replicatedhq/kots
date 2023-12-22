package base

import (
	"io"

	"github.com/pkg/errors"
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/image"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kustomizeimage "sigs.k8s.io/kustomize/api/types"
)

type RewriteImageOptions struct {
	BaseDir           string
	AppSlug           string
	SourceRegistry    registrytypes.RegistryOptions
	DestRegistry      registrytypes.RegistryOptions
	DockerHubRegistry registrytypes.RegistryOptions
	CopyImages        bool
	IsAirgap          bool
	Log               *logger.CLILogger
	ReportWriter      io.Writer
	KotsKinds         *kotsutil.KotsKinds
}

type RewriteImagesResult struct {
	Images        []kustomizeimage.Image          // images to be rewritten
	CheckedImages []kotsv1beta1.InstallationImage // all images found in the installation
}

func RewriteImages(options RewriteImageOptions) (*RewriteImagesResult, error) {
	allImagesPrivate := options.IsAirgap
	additionalImages := make([]string, 0)
	checkedImages := make(map[string]imagetypes.ImageInfo)

	if options.KotsKinds != nil {
		kki, err := kotsutil.GetImagesFromKotsKinds(options.KotsKinds, &options.DestRegistry)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get images from kots kinds")
		}
		additionalImages = kki

		checkedImages = makeImageInfoMap(options.KotsKinds.Installation.Spec.KnownImages)
		if options.KotsKinds.KotsApplication.Spec.ProxyPublicImages {
			allImagesPrivate = true
		}
	}

	newImages, err := image.RewriteImages(options.SourceRegistry, options.DestRegistry, options.AppSlug, options.Log, options.ReportWriter, options.BaseDir, additionalImages, options.CopyImages, allImagesPrivate, checkedImages, options.DockerHubRegistry)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save images")
	}

	return &RewriteImagesResult{
		Images:        newImages,
		CheckedImages: makeInstallationImages(checkedImages),
	}, nil
}
