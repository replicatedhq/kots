package base

import (
	"io"

	"github.com/pkg/errors"
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/image"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/imageutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

type ProcessAirgapImagesOptions struct {
	BaseImages          []string
	KotsKindsImages     []string
	RootDir             string
	AirgapRoot          string
	AirgapBundle        string
	CreateAppDir        bool
	PushImages          bool
	Log                 *logger.CLILogger
	ReplicatedRegistry  registrytypes.RegistryOptions
	ReportWriter        io.Writer
	DestinationRegistry registrytypes.RegistryOptions
	KotsKinds           *kotsutil.KotsKinds
}

type ProcessAirgapImagesResult struct {
	KustomizeImages []kustomizetypes.Image
	KnownImages     []kotsv1beta1.InstallationImage
}

func ProcessAirgapImages(opts ProcessAirgapImagesOptions) (*ProcessAirgapImagesResult, error) {
	pushOpts := imagetypes.PushImagesOptions{
		Registry:       opts.DestinationRegistry,
		Log:            opts.Log,
		ProgressWriter: opts.ReportWriter,
		LogForUI:       true,
	}

	if opts.PushImages {
		if opts.AirgapBundle != "" {
			err := image.TagAndPushAppImagesFromBundle(opts.AirgapBundle, pushOpts)
			if err != nil {
				return nil, errors.Wrap(err, "failed to push images from bundle")
			}
		} else {
			err := image.TagAndPushAppImagesFromPath(opts.AirgapRoot, pushOpts)
			if err != nil {
				return nil, errors.Wrap(err, "failed to push images from dir")
			}
		}
	}

	rewrittenImages := []kustomizetypes.Image{}
	for _, image := range append(opts.BaseImages, opts.KotsKindsImages...) {
		rewrittenImage, err := imageutil.RewriteDockerRegistryImage(opts.DestinationRegistry, image)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to rewrite image %s", image)
		}
		rewrittenImages = append(rewrittenImages, *rewrittenImage)
	}

	withAltNames := make([]kustomizetypes.Image, 0)
	for _, i := range rewrittenImages {
		altNames, err := imageutil.BuildImageAltNames(i)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build image alt names")
		}
		withAltNames = append(withAltNames, altNames...)
	}

	result := &ProcessAirgapImagesResult{
		KustomizeImages: withAltNames,
		// This list is slightly different from the list we get from app specs because of alternative names,
		// but it still works because after rewriting image names with private registry, the lists become the same.
		KnownImages: installationImagesFromKustomizeImages(withAltNames),
	}
	return result, nil
}

func installationImagesFromKustomizeImages(images []kustomizetypes.Image) []kotsv1beta1.InstallationImage {
	result := make([]kotsv1beta1.InstallationImage, 0)
	for _, i := range images {
		result = append(result, kotsv1beta1.InstallationImage{
			Image:     imageutil.SrcImageFromKustomizeImage(i),
			IsPrivate: true,
		})
	}
	return result
}
