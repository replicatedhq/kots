package registry

import (
	"encoding/base64"
	"fmt"
	"path"
	"strings"

	"github.com/pkg/errors"
	kotsregistry "github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/registry/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/collect"
)

// UpdateCollectorSpecsWithRegistryData takes an array of collectors and some environment data (local registry info and license, etc)
// any image that needs to be rewritten to be compatible with the local registry settings or proxy pull
// will be updated and replaced in the spec.  any required image pull secret will be automatically
// inserted into the spec
// an error is returned if anything failed, but the collectors param can always be used after calling (assuming no error)
//
// local registry always overwrites images
// proxy registry only overwrites private images
func UpdateCollectorSpecsWithRegistryData(collectors []*troubleshootv1beta2.Collect, localRegistryInfo types.RegistrySettings, installation kotsv1beta1.Installation, license *kotsv1beta1.License, kotsApplication *kotsv1beta1.Application) ([]*troubleshootv1beta2.Collect, error) {
	if localRegistryInfo.IsValid() {
		updatedCollectors, err := updateCollectorsWithLocalRegistryData(collectors, localRegistryInfo, installation, license)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update collectors with local registry info")
		}

		return updatedCollectors, nil
	}

	updatedCollectors, err := updateCollectorsWithProxyRegistryData(collectors, localRegistryInfo, installation, license, kotsApplication)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update collectors with replicated registry info")
	}

	return updatedCollectors, nil
}

func updateCollectorsWithLocalRegistryData(collectors []*troubleshootv1beta2.Collect, localRegistryInfo types.RegistrySettings, installation kotsv1beta1.Installation, license *kotsv1beta1.License) ([]*troubleshootv1beta2.Collect, error) {
	updatedCollectors := []*troubleshootv1beta2.Collect{}

	makeImagePullSecret := func(namespace string) (*troubleshootv1beta2.ImagePullSecrets, error) {
		pullSecrets, err := kotsregistry.PullSecretForRegistries([]string{localRegistryInfo.Hostname}, localRegistryInfo.Username, localRegistryInfo.Password, namespace, "")
		if err != nil {
			return nil, err
		}
		imagePullSecret := &troubleshootv1beta2.ImagePullSecrets{
			SecretType: "kubernetes.io/dockerconfigjson",
			Data: map[string]string{
				".dockerconfigjson": base64.StdEncoding.EncodeToString(pullSecrets.AdminConsoleSecret.Data[".dockerconfigjson"]),
			},
		}

		return imagePullSecret, nil
	}

	for _, c := range collectors {
		collector := troubleshootv1beta2.GetCollector(c)
		if collector == nil {
			continue
		}

		if imageRunner, ok := collector.(collect.ImageRunner); ok {
			newImage := rewriteImage(localRegistryInfo.Hostname, localRegistryInfo.Namespace, imageRunner.GetImage())
			imageRunner.SetImage(newImage)

			imagePullSecret, err := makeImagePullSecret(imageRunner.GetNamespace())
			if err != nil {
				return nil, errors.Wrap(err, "failed to generate pull secret for image runner")
			}
			imageRunner.SetImagePullSecret(imagePullSecret)
		} else if podSpecRunner, ok := collector.(collect.PodSpecRunner); ok {
			imagePullSecret, err := makeImagePullSecret(podSpecRunner.GetNamespace())
			if err != nil {
				return nil, errors.Wrap(err, "failed to generate pull secret for pod runner")
			}
			podSpecRunner.SetImagePullSecret(imagePullSecret)

			podSpec := podSpecRunner.GetPodSpec()
			for i := range podSpec.InitContainers {
				podSpec.InitContainers[i].Image = rewriteImage(localRegistryInfo.Hostname, localRegistryInfo.Namespace, podSpec.InitContainers[i].Image)
			}
			for i := range podSpec.Containers {
				podSpec.Containers[i].Image = rewriteImage(localRegistryInfo.Hostname, localRegistryInfo.Namespace, podSpec.Containers[i].Image)
			}
		} else if c.RegistryImages != nil {
			imagePullSecret, err := makeImagePullSecret(c.RegistryImages.Namespace)
			if err != nil {
				return nil, errors.Wrap(err, "failed to generate pull secret for registry images collector")
			}
			c.RegistryImages.ImagePullSecrets = imagePullSecret

			images := []string{}
			for _, knownImage := range installation.Spec.KnownImages {
				image := rewriteImage(localRegistryInfo.Hostname, localRegistryInfo.Namespace, knownImage.Image)
				images = append(images, image)
			}
			c.RegistryImages.Images = images
		}
		updatedCollectors = append(updatedCollectors, c)
	}

	return updatedCollectors, nil
}

func updateCollectorsWithProxyRegistryData(collectors []*troubleshootv1beta2.Collect, localRegistryInfo types.RegistrySettings, installation kotsv1beta1.Installation, license *kotsv1beta1.License, kotsApplication *kotsv1beta1.Application) ([]*troubleshootv1beta2.Collect, error) {
	updatedCollectors := []*troubleshootv1beta2.Collect{}

	registryProxyInfo := kotsregistry.GetRegistryProxyInfo(license, &installation, kotsApplication)

	makeImagePullSecret := func(namespace string) (*troubleshootv1beta2.ImagePullSecrets, error) {
		pullSecrets, err := kotsregistry.PullSecretForRegistries(registryProxyInfo.ToSlice(), license.Spec.LicenseID, license.Spec.LicenseID, namespace, "")
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate pull secret for proxy registry")
		}
		imagePullSecret := &troubleshootv1beta2.ImagePullSecrets{
			SecretType: "kubernetes.io/dockerconfigjson",
			Data: map[string]string{
				".dockerconfigjson": base64.StdEncoding.EncodeToString(pullSecrets.AdminConsoleSecret.Data[".dockerconfigjson"]),
			},
		}

		return imagePullSecret, nil
	}

	rewrite := func(image string) string {
		registryHost := strings.Split(image, "/")[0]
		if registryHost == registryProxyInfo.Registry {
			// if it's the replicated registry, no change, just add image pull secret
			return image
		}
		tag := strings.Split(image, ":")
		image = kotsregistry.MakeProxiedImageURL(registryProxyInfo.Proxy, license.Spec.AppSlug, image)
		if len(tag) > 1 {
			image = fmt.Sprintf("%s:%s", image, tag[len(tag)-1])
		}
		return image
	}

	// for all known private images, rewrite to the replicated proxy and add license image pull secret
	for _, c := range collectors {
		collector := troubleshootv1beta2.GetCollector(c)
		if collector == nil {
			continue
		}

		// all collectors that include images in the spec should have an if / else statement here
		if imageRunner, ok := collector.(collect.ImageRunner); ok {
			for _, knownImage := range installation.Spec.KnownImages {
				image := imageRunner.GetImage()
				if knownImage.Image != image || !knownImage.IsPrivate {
					continue
				}

				imageRunner.SetImage(rewrite(image))
				imagePullSecret, err := makeImagePullSecret(imageRunner.GetNamespace())
				if err != nil {
					return nil, errors.Wrap(err, "failed to generate pull secret for image runner")
				}
				imageRunner.SetImagePullSecret(imagePullSecret)
			}
		} else if podsSpecRunner, ok := collector.(collect.PodSpecRunner); ok {
			podSpec := podsSpecRunner.GetPodSpec()
			for _, knownImage := range installation.Spec.KnownImages {
				for i, container := range podSpec.InitContainers {
					if knownImage.Image != container.Image || !knownImage.IsPrivate {
						continue
					}
					podSpec.InitContainers[i].Image = rewrite(container.Image)
				}
				for i, container := range podSpec.Containers {
					if knownImage.Image != container.Image || !knownImage.IsPrivate {
						continue
					}
					podSpec.Containers[i].Image = rewrite(container.Image)
				}
			}
			imagePullSecret, err := makeImagePullSecret(podsSpecRunner.GetNamespace())
			if err != nil {
				return nil, errors.Wrap(err, "failed to generate pull secret for image runner")
			}
			podsSpecRunner.SetImagePullSecret(imagePullSecret)
		}

		updatedCollectors = append(updatedCollectors, c)
	}
	return updatedCollectors, nil
}

func rewriteImage(newHost string, newNamespace string, image string) string {
	imageParts := strings.Split(image, "/")
	imageNameWithOptionalTag := imageParts[len(imageParts)-1]

	return path.Join(newHost, newNamespace, imageNameWithOptionalTag)
}
