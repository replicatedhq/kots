package version

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/distribution/reference"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
)

type ImageRewriteFunc func(upstreamImage string, alwaysRewrite bool) (image string, imagePullSecrets []corev1.LocalObjectReference, err error)

// return "alpha" for all invalid versions of kots,
// kotsadm tag that matches this version for others
func KotsadmTag(registryConfig types.RegistryConfig) string {
	if registryConfig.OverrideVersion != "" {
		return registryConfig.OverrideVersion
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

func KotsadmRegistry(registryConfig types.RegistryConfig) string {
	if registryConfig.OverrideRegistry == "" {
		// Images hosted in docker hub
		if registryConfig.OverrideNamespace == "" {
			return "kotsadm"
		} else {
			return registryConfig.OverrideNamespace
		}
	}

	if registryConfig.OverrideNamespace == "" {
		return registryConfig.OverrideRegistry
	}

	return fmt.Sprintf("%s/%s", registryConfig.OverrideRegistry, registryConfig.OverrideNamespace)
}

func KotsadmPullSecret(namespace string, registryConfig types.RegistryConfig) *corev1.Secret {
	if registryConfig.OverrideRegistry == "" {
		return nil
	}

	secrets, _ := registry.PullSecretForRegistries([]string{registryConfig.OverrideRegistry}, registryConfig.Username, registryConfig.Password, namespace, "")

	secret := secrets.AdminConsoleSecret
	secret.ObjectMeta.Name = types.PrivateKotsadmRegistrySecret
	secret.ObjectMeta.Labels = types.GetKotsadmLabels()

	return secret
}

// This function will rewrite images and use the version from this binary as image tag when not overriden
func KotsadmImageRewriteKotsadmRegistry(namespace string, registryConfig *types.RegistryConfig) ImageRewriteFunc {
	secret := KotsadmPullSecret(namespace, *registryConfig)

	return func(upstreamImage string, alwaysRewrite bool) (image string, imagePullSecrets []corev1.LocalObjectReference, err error) {
		image = upstreamImage

		if registryConfig == nil {
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
		image = fmt.Sprintf("%s/%s:%s", KotsadmRegistry(*registryConfig), imageName, KotsadmTag(*registryConfig))

		if secret != nil {
			imagePullSecrets = []corev1.LocalObjectReference{
				{Name: secret.ObjectMeta.Name},
			}
		}
		return image, imagePullSecrets, err
	}
}

// This function will rewrite images and use the image's original tag
func DependencyImageRewriteKotsadmRegistry(namespace string, registryConfig *types.RegistryConfig) ImageRewriteFunc {
	secret := KotsadmPullSecret(namespace, *registryConfig)

	return func(upstreamImage string, alwaysRewrite bool) (image string, imagePullSecrets []corev1.LocalObjectReference, err error) {
		image = upstreamImage

		if registryConfig == nil {
			return image, imagePullSecrets, err
		}

		if !alwaysRewrite && secret == nil {
			return image, imagePullSecrets, err
		}

		named, err := reference.ParseNormalizedNamed(upstreamImage)
		if err != nil {
			return image, imagePullSecrets, err
		}

		tag := ""
		if tagged, ok := named.(reference.Tagged); ok {
			tag = tagged.Tag()
			// TODO: support digests
			// else if can, ok := named.(reference.Canonical); ok {
			// 	tag = can.Digest().String()
		} else {
			return image, imagePullSecrets, errors.Errorf("only tagged references can be rewriten: %s", image)
		}

		parts := strings.Split(reference.Path(named), "/")
		imageName := parts[len(parts)-1] // why not include the namespace here?
		image = fmt.Sprintf("%s/%s:%s", KotsadmRegistry(*registryConfig), imageName, tag)

		if secret != nil {
			imagePullSecrets = []corev1.LocalObjectReference{
				{Name: secret.ObjectMeta.Name},
			}
		}
		return image, imagePullSecrets, err
	}
}
