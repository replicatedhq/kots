package base

import (
	"strings"

	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/distribution/distribution/v3/reference"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/imageutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

type RewriteImagesOptions struct {
	Images              []string
	DestinationRegistry dockerregistrytypes.RegistryOptions
}

// RewriteImages rewrites all images to point to the configured destination registry.
func RewriteImages(images []string, destRegistry dockerregistrytypes.RegistryOptions) ([]kustomizetypes.Image, error) {
	rewrittenImages := []kustomizetypes.Image{}
	rewritten := map[string]bool{}

	for _, image := range images {
		if _, ok := rewritten[image]; ok {
			continue
		}
		rewrittenImage, err := imageutil.RewriteDockerRegistryImage(destRegistry, image)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to rewrite image %s", image)
		}
		rewrittenImages = append(rewrittenImages, *rewrittenImage)
		rewritten[image] = true
	}

	withAltNames := make([]kustomizetypes.Image, 0)
	for _, i := range rewrittenImages {
		altNames, err := imageutil.BuildImageAltNames(i)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build image alt names")
		}
		withAltNames = append(withAltNames, altNames...)
	}

	return withAltNames, nil
}

// RewritePrivateImages rewrites private images to be proxied through proxy.replicated.com,
// and rewrites replicated registry images to use the custom registry domain if configured
func RewritePrivateImages(images []string, kotsKinds *kotsutil.KotsKinds, license *kotsv1beta1.License) ([]kustomizetypes.Image, error) {
	replicatedRegistryInfo := registry.GetRegistryProxyInfo(license, &kotsKinds.Installation, &kotsKinds.KotsApplication)

	replicatedRegistry := dockerregistrytypes.RegistryOptions{
		Endpoint:         replicatedRegistryInfo.Registry,
		ProxyEndpoint:    replicatedRegistryInfo.Proxy,
		UpstreamEndpoint: replicatedRegistryInfo.Upstream,
	}

	installationImages := make(map[string]imagetypes.InstallationImageInfo)
	for _, i := range kotsKinds.Installation.Spec.KnownImages {
		installationImages[i.Image] = imagetypes.InstallationImageInfo{
			IsPrivate: i.IsPrivate,
		}
	}

	privateImages := []string{}
	for _, img := range images {
		if installationImages[img].IsPrivate {
			privateImages = append(privateImages, img)
		}
	}

	kustomizeImages := make([]kustomizetypes.Image, 0)
	for _, privateImage := range privateImages {
		dockerRef, err := dockerref.ParseDockerRef(privateImage)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse docker ref %q", privateImage)
		}

		registryHost := dockerref.Domain(dockerRef)
		if registryHost == replicatedRegistry.Endpoint {
			// replicated images are also private, but we don't rewrite those
			continue
		}

		image := kustomizetypes.Image{}
		if registryHost == replicatedRegistry.UpstreamEndpoint {
			// image is using the upstream replicated registry, but a custom registry domain is configured, so rewrite to use the custom domain
			image = kustomizetypes.Image{
				Name:    dockerRef.Name(),
				NewName: strings.Replace(dockerRef.Name(), registryHost, replicatedRegistry.Endpoint, 1),
			}
		} else {
			// all other private images are rewritten to use the replicated proxy
			image = kustomizetypes.Image{
				Name:    dockerRef.Name(),
				NewName: registry.MakeProxiedImageURL(replicatedRegistry.ProxyEndpoint, license.Spec.AppSlug, privateImage),
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

	return kustomizeImages, nil
}
