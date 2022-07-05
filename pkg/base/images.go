package base

import (
	"fmt"

	imagedocker "github.com/containers/image/v5/docker"
	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	kotsimage "github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	kustomizeimage "sigs.k8s.io/kustomize/api/types"
)

type FindPrivateImagesOptions struct {
	BaseDir            string
	AppSlug            string
	ReplicatedRegistry registry.RegistryOptions
	DockerHubRegistry  registry.RegistryOptions
	Installation       *kotsv1beta1.Installation
	AllImagesPrivate   bool
	HelmChartPath      string
	UseHelmInstall     map[string]bool
}

type FindPrivateImagesResult struct {
	Images        []kustomizeimage.Image          // images to be rewritten
	Docs          []k8sdoc.K8sDoc                 // docs that have rewritten images
	CheckedImages []kotsv1beta1.InstallationImage // all images found in the installation
}

func FindPrivateImages(options FindPrivateImagesOptions) (*FindPrivateImagesResult, error) {
	checkedImages := makeImageInfoMap(options.Installation.Spec.KnownImages)
	upstreamImages, objects, err := image.GetPrivateImages(options.BaseDir, checkedImages, options.AllImagesPrivate, options.DockerHubRegistry, options.HelmChartPath, options.UseHelmInstall)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list upstream images")
	}

	kustomizeImages := make([]kustomizeimage.Image, 0)
	for _, upstreamImage := range upstreamImages {
		// ParseReference requires the // prefix
		ref, err := imagedocker.ParseReference(fmt.Sprintf("//%s", upstreamImage))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse image ref %q", upstreamImage)
		}
		dockerRef := ref.DockerReference()

		registryHost := dockerref.Domain(dockerRef)
		if registryHost == options.ReplicatedRegistry.Endpoint {
			// replicated images are also private, but we don't rewrite those
			continue
		}

		image := kustomizeimage.Image{
			Name:    dockerRef.Name(),
			NewName: registry.MakeProxiedImageURL(options.ReplicatedRegistry.ProxyEndpoint, options.AppSlug, upstreamImage),
		}

		if tagged, ok := dockerRef.(reference.Tagged); ok {
			image.NewTag = tagged.Tag()
		} else if can, ok := dockerRef.(reference.Canonical); ok {
			image.NewTag = can.Digest().String()
		} else {
			image.NewTag = "latest"
		}

		kustomizeImages = append(kustomizeImages, kotsimage.BuildImageAltNames(image)...)
	}

	return &FindPrivateImagesResult{
		Images:        kustomizeImages,
		Docs:          objects,
		CheckedImages: makeInstallationImages(checkedImages),
	}, nil
}

func makeImageInfoMap(images []kotsv1beta1.InstallationImage) map[string]image.ImageInfo {
	result := make(map[string]image.ImageInfo)
	for _, i := range images {
		result[i.Image] = image.ImageInfo{
			IsPrivate: i.IsPrivate,
		}
	}
	return result
}

func makeInstallationImages(images map[string]image.ImageInfo) []kotsv1beta1.InstallationImage {
	result := make([]kotsv1beta1.InstallationImage, 0)
	for image, info := range images {
		result = append(result, kotsv1beta1.InstallationImage{
			Image:     image,
			IsPrivate: info.IsPrivate,
		})
	}
	return result
}
