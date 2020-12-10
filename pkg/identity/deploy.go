package identity

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	KotsadmNamePrefix = "kotsadm"
)

func ImageRewriteKotsadmRegistry(namespace string, registryOptions *kotsadmtypes.KotsadmOptions) identitydeploy.ImageRewriteFunc {
	if registryOptions == nil {
		return nil
	}

	secret := kotsadmversion.KotsadmPullSecret(namespace, *registryOptions)
	if secret == nil {
		return nil
	}

	return func(upstreamImage, path, tag string) (string, []corev1.LocalObjectReference) {
		parts := strings.Split(path, "/")
		imageName := parts[len(parts)-1] // why not include the whole path here?

		image := fmt.Sprintf("%s/%s:%s", kotsadmversion.KotsadmRegistry(*registryOptions), imageName, kotsadmversion.KotsadmTag(*registryOptions))
		imagePullSecrets := []corev1.LocalObjectReference{
			{Name: secret.ObjectMeta.Name},
		}
		return image, imagePullSecrets
	}
}

func Deploy(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig, registryOptions *kotsadmtypes.KotsadmOptions) error {
	dexConfig, err := getDexConfig(ctx, clientset, namespace, identityConfig.Spec, ingressConfig.Spec)
	if err != nil {
		return errors.Wrap(err, "failed to get dex config")
	}

	fn := ImageRewriteKotsadmRegistry(namespace, registryOptions)
	return identitydeploy.Deploy(ctx, clientset, namespace, KotsadmNamePrefix, dexConfig, identityConfig.Spec.IngressConfig, fn)
}

func Configure(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig) error {
	dexConfig, err := getDexConfig(ctx, clientset, namespace, identityConfig.Spec, ingressConfig.Spec)
	if err != nil {
		return errors.Wrap(err, "failed to get dex config")
	}

	return identitydeploy.Configure(ctx, clientset, namespace, KotsadmNamePrefix, dexConfig)
}

func Render(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig, registryOptions *kotsadmtypes.KotsadmOptions) (map[string][]byte, error) {
	dexConfig, err := getDexConfig(ctx, clientset, namespace, identityConfig.Spec, ingressConfig.Spec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dex config")
	}

	fn := ImageRewriteKotsadmRegistry(namespace, registryOptions)
	return identitydeploy.Render(ctx, KotsadmNamePrefix, dexConfig, identityConfig.Spec.IngressConfig, fn)
}

func Undeploy(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	return identitydeploy.Undeploy(ctx, clientset, namespace, KotsadmNamePrefix)
}
