package base

import (
	"strings"

	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/distribution/distribution/v3/reference"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/image"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/imageutil"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kustomizeimage "sigs.k8s.io/kustomize/api/types"
)

type FindPrivateImagesOptions struct {
	BaseDir            string
	AppSlug            string
	ReplicatedRegistry registrytypes.RegistryOptions
	DockerHubRegistry  registrytypes.RegistryOptions
	Installation       *kotsv1beta1.Installation
	AllImagesPrivate   bool
	HelmChartPath      string
	UseHelmInstall     map[string]bool
	KotsKindsImages    []string
}

type FindPrivateImagesResult struct {
	Images        []kustomizeimage.Image          // images to be rewritten
	Docs          []k8sdoc.K8sDoc                 // docs that have rewritten images
	CheckedImages []kotsv1beta1.InstallationImage // all images found in the installation
}

func FindPrivateImages(options FindPrivateImagesOptions) (*FindPrivateImagesResult, error) {
	checkedImages := makeImageInfoMap(options.Installation.Spec.KnownImages)
	upstreamImages, objects, err := image.GetPrivateImages(options.BaseDir, options.KotsKindsImages, checkedImages, options.AllImagesPrivate, options.DockerHubRegistry, options.HelmChartPath, options.UseHelmInstall)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list upstream images")
	}

	kustomizeImages := make([]kustomizeimage.Image, 0)
	for _, upstreamImage := range upstreamImages {
		dockerRef, err := dockerref.ParseDockerRef(upstreamImage)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse docker ref %q", upstreamImage)
		}

		registryHost := dockerref.Domain(dockerRef)
		if registryHost == options.ReplicatedRegistry.Endpoint {
			// replicated images are also private, but we don't rewrite those
			continue
		}

		image := kustomizeimage.Image{}
		if registryHost == options.ReplicatedRegistry.UpstreamEndpoint {
			// image is using the upstream replicated registry, but a custom registry domain is configured, so rewrite to use the custom domain
			image = kustomizeimage.Image{
				Name:    dockerRef.Name(),
				NewName: strings.Replace(dockerRef.Name(), registryHost, options.ReplicatedRegistry.Endpoint, 1),
			}
		} else {
			// all other private images are rewritten to use the replicated proxy
			image = kustomizeimage.Image{
				Name:    dockerRef.Name(),
				NewName: registry.MakeProxiedImageURL(options.ReplicatedRegistry.ProxyEndpoint, options.AppSlug, upstreamImage),
			}
		}

		if can, ok := dockerRef.(reference.Canonical); ok {
			image.Digest = can.Digest().String()
		} else if tagged, ok := dockerRef.(reference.Tagged); ok {
			image.NewTag = tagged.Tag()
		} else {
			image.NewTag = "latest"
		}

		altNames, err := imageutil.BuildImageAltNames(image)
		if err != nil {
			return nil, errors.Wrap(err, "failed build alt names")
		}
		kustomizeImages = append(kustomizeImages, altNames...)
	}

	return &FindPrivateImagesResult{
		Images:        kustomizeImages,
		Docs:          objects,
		CheckedImages: makeInstallationImages(checkedImages),
	}, nil
}

func makeImageInfoMap(images []kotsv1beta1.InstallationImage) map[string]imagetypes.ImageInfo {
	result := make(map[string]imagetypes.ImageInfo)
	for _, i := range images {
		result[i.Image] = imagetypes.ImageInfo{
			IsPrivate: i.IsPrivate,
		}
	}
	return result
}

func makeInstallationImages(images map[string]imagetypes.ImageInfo) []kotsv1beta1.InstallationImage {
	result := make([]kotsv1beta1.InstallationImage, 0)
	for image, info := range images {
		result = append(result, kotsv1beta1.InstallationImage{
			Image:     image,
			IsPrivate: info.IsPrivate,
		})
	}
	return result
}
