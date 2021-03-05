package version

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/docker/distribution/reference"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
)

type ImageRewriteFunc func(upstreamImage string, alwaysRewrite bool) (image string, imagePullSecrets []corev1.LocalObjectReference, err error)

// return "alpha" for all invalid versions of kots,
// kotsadm tag that matches this version for others
func KotsadmTag(options types.KotsadmOptions) string {
	if options.OverrideVersion != "" {
		return options.OverrideVersion
	}

	return KotsadmTagForVersionString(buildversion.Version())
}

func KotsadmTagForVersionString(kotsVersion string) string {
	version, err := semver.NewVersion(kotsVersion)
	if err != nil {
		return "alpha"
	}

	if strings.Contains(version.Prerelease(), "dirty") {
		return "alpha"
	}

	if !strings.HasPrefix(kotsVersion, "v") {
		kotsVersion = fmt.Sprintf("v%s", kotsVersion)
	}

	return kotsVersion
}

func KotsadmRegistry(options types.KotsadmOptions) string {
	if options.OverrideRegistry == "" {
		if options.OverrideNamespace == "" {
			return "kotsadm"
		} else {
			return options.OverrideNamespace
		}
	}

	registry := options.OverrideRegistry
	namespace := options.OverrideNamespace

	hostParts := strings.Split(options.OverrideRegistry, "/")
	if len(hostParts) == 2 {
		registry = hostParts[0]
		namespace = hostParts[1]
	}

	if namespace == "" {
		// note that this makes it impossible to have a registry without a namespace
		// keeping for backwards compatibility
		return fmt.Sprintf("%s/kotsadm", registry)
	}

	return fmt.Sprintf("%s/%s", registry, namespace)
}

func KotsadmPullSecret(namespace string, options types.KotsadmOptions) *corev1.Secret {
	if options.OverrideRegistry == "" || options.Username == "" || options.Password == "" {
		return nil
	}

	secret, _ := registry.PullSecretForRegistries([]string{options.OverrideRegistry}, options.Username, options.Password, namespace)
	if secret == nil {
		return nil
	}

	secret.ObjectMeta.Name = types.PrivateKotsadmRegistrySecret
	secret.ObjectMeta.Labels = types.GetKotsadmLabels()

	return secret
}

func ImageRewriteKotsadmRegistry(namespace string, registryOptions *types.KotsadmOptions) ImageRewriteFunc {
	secret := KotsadmPullSecret(namespace, *registryOptions)

	return func(upstreamImage string, alwaysRewrite bool) (image string, imagePullSecrets []corev1.LocalObjectReference, err error) {
		image = upstreamImage

		if registryOptions == nil {
			return image, imagePullSecrets, err
		}

		if !alwaysRewrite && secret == nil {
			return image, imagePullSecrets, err
		}

		named, err := reference.ParseNormalizedNamed(upstreamImage)
		if err != nil {
			return image, imagePullSecrets, err
		}

		parts := strings.Split(reference.Path(named), "/")
		imageName := parts[len(parts)-1] // why not include the namespace here?
		image = fmt.Sprintf("%s/%s:%s", KotsadmRegistry(*registryOptions), imageName, KotsadmTag(*registryOptions))

		if secret != nil {
			imagePullSecrets = []corev1.LocalObjectReference{
				{Name: secret.ObjectMeta.Name},
			}
		}
		return image, imagePullSecrets, err
	}
}
