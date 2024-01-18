package base

import (
	"sort"
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
	BaseImages         []string
	KotsKindsImages    []string
	AppSlug            string
	ReplicatedRegistry registrytypes.RegistryOptions
	DockerHubRegistry  registrytypes.RegistryOptions
	Installation       *kotsv1beta1.Installation
	AllImagesPrivate   bool
}

type FindPrivateImagesResult struct {
	Images        []kustomizeimage.Image          // images to be rewritten
	CheckedImages []kotsv1beta1.InstallationImage // all images found in the installation
}

func FindImages(b *Base) ([]string, []k8sdoc.K8sDoc, error) {
	uniqueImages := make(map[string]bool)
	objectsWithImages := make([]k8sdoc.K8sDoc, 0) // all objects where images are referenced from

	for _, file := range b.Files {
		parsed, err := k8sdoc.ParseYAML(file.Content)
		if err != nil {
			continue
		}

		images := parsed.ListImages()
		if len(images) > 0 {
			objectsWithImages = append(objectsWithImages, parsed)
		}

		for _, image := range images {
			uniqueImages[image] = true
		}
	}

	for _, subBase := range b.Bases {
		subImages, subObjects, err := FindImages(&subBase)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to find images in sub base %s", subBase.Path)
		}

		objectsWithImages = append(objectsWithImages, subObjects...)

		for _, subImage := range subImages {
			uniqueImages[subImage] = true
		}
	}

	result := make([]string, 0, len(uniqueImages))
	for i := range uniqueImages {
		result = append(result, i)
	}
	sort.Strings(result) // sort the images to get an ordered and reproducible output for easier testing

	return result, objectsWithImages, nil
}

func FindPrivateImages(opts FindPrivateImagesOptions) (*FindPrivateImagesResult, error) {
	checkedImages := makeInstallationImageInfoMap(opts.Installation.Spec.KnownImages)
	privateImages, err := image.GetPrivateImages(opts.BaseImages, opts.KotsKindsImages, checkedImages, opts.AllImagesPrivate, opts.DockerHubRegistry)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list upstream images")
	}

	kustomizeImages := make([]kustomizeimage.Image, 0)
	for _, privateImage := range privateImages {
		dockerRef, err := dockerref.ParseDockerRef(privateImage)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse docker ref %q", privateImage)
		}

		registryHost := dockerref.Domain(dockerRef)
		if registryHost == opts.ReplicatedRegistry.Endpoint {
			// replicated images are also private, but we don't rewrite those
			continue
		}

		image := kustomizeimage.Image{}
		if registryHost == opts.ReplicatedRegistry.UpstreamEndpoint {
			// image is using the upstream replicated registry, but a custom registry domain is configured, so rewrite to use the custom domain
			image = kustomizeimage.Image{
				Name:    dockerRef.Name(),
				NewName: strings.Replace(dockerRef.Name(), registryHost, opts.ReplicatedRegistry.Endpoint, 1),
			}
		} else {
			// all other private images are rewritten to use the replicated proxy
			image = kustomizeimage.Image{
				Name:    dockerRef.Name(),
				NewName: registry.MakeProxiedImageURL(opts.ReplicatedRegistry.ProxyEndpoint, opts.AppSlug, privateImage),
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
		CheckedImages: installationImagesFromInfoMap(checkedImages),
	}, nil
}

func makeInstallationImageInfoMap(images []kotsv1beta1.InstallationImage) map[string]imagetypes.InstallationImageInfo {
	result := make(map[string]imagetypes.InstallationImageInfo)
	for _, i := range images {
		result[i.Image] = imagetypes.InstallationImageInfo{
			IsPrivate: i.IsPrivate,
		}
	}
	return result
}

func installationImagesFromInfoMap(images map[string]imagetypes.InstallationImageInfo) []kotsv1beta1.InstallationImage {
	result := make([]kotsv1beta1.InstallationImage, 0)
	for image, info := range images {
		result = append(result, kotsv1beta1.InstallationImage{
			Image:     image,
			IsPrivate: info.IsPrivate,
		})
	}
	return result
}
