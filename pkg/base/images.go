package base

import (
	"fmt"

	imagedocker "github.com/containers/image/docker"
	dockerref "github.com/containers/image/docker/reference"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	kustomizeimage "sigs.k8s.io/kustomize/v3/pkg/image"
)

type FindPrivateImagesOptions struct {
	BaseDir            string
	AppSlug            string
	ReplicatedRegistry registry.RegistryOptions
}

func FindPrivateImages(options FindPrivateImagesOptions) ([]kustomizeimage.Image, []*k8sdoc.Doc, error) {
	upstreamImages, objects, err := image.GetPrivateImages(options.BaseDir)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to list upstream images")
	}

	result := make([]kustomizeimage.Image, 0)
	for _, upstreamImage := range upstreamImages {
		// ParseReference requires the // prefix
		ref, err := imagedocker.ParseReference(fmt.Sprintf("//%s", upstreamImage))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to parse image ref:%s", upstreamImage)
		}

		registryHost := dockerref.Domain(ref.DockerReference())
		if registryHost == options.ReplicatedRegistry.Endpoint {
			// replicated images are also private, but we don't rewrite those
			continue
		}

		image := kustomizeimage.Image{
			Name:    upstreamImage,
			NewName: registry.MakeProxiedImageURL(options.ReplicatedRegistry.ProxyEndpoint, options.AppSlug, upstreamImage),
		}
		result = append(result, image)
	}

	return result, objects, nil
}

type FindObjectsWithImagesOptions struct {
	BaseDir string
}

func FindObjectsWithImages(options FindObjectsWithImagesOptions) ([]*k8sdoc.Doc, error) {
	objects, err := image.GetObjectsWithImages(options.BaseDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list upstream images")
	}

	return objects, nil
}
