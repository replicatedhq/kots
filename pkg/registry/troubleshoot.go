package registry

import (
	"encoding/base64"
	"fmt"
	"path"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsregistry "github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/registry/types"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
)

// UpdateCollectorSpecsWithRegistryData takes an array of collectors and some environment data (local registry info and license, etc)
// any image that needs to be rewritten to be compatible with the local registry settings or proxy pull
// will be updated and replaced in the spec.  any required image pull secret will be automatically
// inserted into the spec
// an error is returned if anything failed, but the collectors param can always be used after calling (assuming no error)
func UpdateCollectorSpecsWithRegistryData(collectors []*troubleshootv1beta2.Collect, localRegistryInfo types.RegistrySettings, knownImages []kotsv1beta1.InstallationImage, license *kotsv1beta1.License) ([]*troubleshootv1beta2.Collect, error) {
	// if there's a local registry, always attach that image pull secret for all, and
	// always rewrite
	updatedCollectors := make([]*troubleshootv1beta2.Collect, len(collectors))

	if localRegistryInfo.IsValid() {
		for idx, collect := range collectors {
			if collect.Run != nil {
				run := collect.Run

				run.Image = rewriteImage(localRegistryInfo.Hostname, localRegistryInfo.Namespace, run.Image)
				pullSecrets, err := kotsregistry.PullSecretForRegistries([]string{localRegistryInfo.Hostname}, localRegistryInfo.Username, localRegistryInfo.Password, run.Namespace, "")
				if err != nil {
					return nil, errors.Wrap(err, "failed to generate pull secret for registry")
				}

				run.ImagePullSecret = &troubleshootv1beta2.ImagePullSecrets{
					SecretType: "kubernetes.io/dockerconfigjson",
					Data: map[string]string{
						".dockerconfigjson": base64.StdEncoding.EncodeToString(pullSecrets.AdminConsoleSecret.Data[".dockerconfigjson"]),
					},
				}
				collect.Run = run

				updatedCollectors[idx] = collect
			} else if collect.RegistryImages != nil {
				pullSecrets, err := kotsregistry.PullSecretForRegistries([]string{localRegistryInfo.Hostname}, localRegistryInfo.Username, localRegistryInfo.Password, collect.RegistryImages.Namespace, "")
				if err != nil {
					return nil, errors.Wrap(err, "failed to generate pull secret for registry")
				}

				collect.RegistryImages.ImagePullSecrets = &troubleshootv1beta2.ImagePullSecrets{
					SecretType: "kubernetes.io/dockerconfigjson",
					Data: map[string]string{
						".dockerconfigjson": base64.StdEncoding.EncodeToString(pullSecrets.AdminConsoleSecret.Data[".dockerconfigjson"]),
					},
				}

				images := []string{}
				for _, knownImage := range knownImages {
					image := rewriteImage(localRegistryInfo.Hostname, localRegistryInfo.Namespace, knownImage.Image)
					images = append(images, image)
				}
				collect.RegistryImages.Images = images

				updatedCollectors[idx] = collect

			} else {
				updatedCollectors[idx] = collect
			}
		}

		return updatedCollectors, nil
	}

	registryProxyInfo := kotsregistry.ProxyEndpointFromLicense(license)

	// for all known private images, rewrite to the replicated proxy and add license image pull secret
	for idx, collect := range collectors {
		// all collectors that include images in the spec should have an if / else statement here
		if collect.Run != nil {
			for _, knownImage := range knownImages {
				if knownImage.Image == collect.Run.Image && knownImage.IsPrivate {
					run := collect.Run

					// if it's the replicated registry, no change, just add image pull secret
					registryHost := strings.Split(run.Image, "/")[0]
					if registryHost != registryProxyInfo.Registry {
						tag := strings.Split(run.Image, ":")
						run.Image = kotsregistry.MakeProxiedImageURL(registryProxyInfo.Proxy, license.Spec.AppSlug, run.Image)
						if len(tag) > 1 {
							run.Image = fmt.Sprintf("%s:%s", run.Image, tag[len(tag)-1])
						}
						pullSecrets, err := kotsregistry.PullSecretForRegistries([]string{registryProxyInfo.Proxy}, license.Spec.LicenseID, license.Spec.LicenseID, run.Namespace, "")
						if err != nil {
							return nil, errors.Wrap(err, "failed to generate pull secret for proxy registry")
						}

						run.ImagePullSecret = &troubleshootv1beta2.ImagePullSecrets{
							SecretType: "kubernetes.io/dockerconfigjson",
							Data: map[string]string{
								".dockerconfigjson": base64.StdEncoding.EncodeToString(pullSecrets.AdminConsoleSecret.Data[".dockerconfigjson"]),
							},
						}

						collect.Run = run
					} else {
						pullSecrets, err := kotsregistry.PullSecretForRegistries([]string{registryProxyInfo.Registry}, license.Spec.LicenseID, license.Spec.LicenseID, run.Namespace, "")
						if err != nil {
							return nil, errors.Wrap(err, "failed to generate pull secret for replicated registry")
						}

						run.ImagePullSecret = &troubleshootv1beta2.ImagePullSecrets{
							SecretType: "kubernetes.io/dockerconfigjson",
							Data: map[string]string{
								".dockerconfigjson": base64.StdEncoding.EncodeToString(pullSecrets.AdminConsoleSecret.Data[".dockerconfigjson"]),
							},
						}

						collect.Run = run
					}

					collectors[idx].Run = run
				}
			}

			updatedCollectors[idx] = collect
		} else {
			updatedCollectors[idx] = collect
		}
	}
	return updatedCollectors, nil
}

func rewriteImage(newHost string, newNamespace string, image string) string {
	imageParts := strings.Split(image, "/")
	imageNameWithOptionalTag := imageParts[len(imageParts)-1]

	return path.Join(newHost, newNamespace, imageNameWithOptionalTag)
}
