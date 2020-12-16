package identity

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	"github.com/replicatedhq/kots/pkg/ingress"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	KotsadmNamePrefix = "kotsadm"
)

func Deploy(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig, registryOptions *kotsadmtypes.KotsadmOptions, proxyEnv map[string]string) error {
	identityConfig.Spec.ClientID = "kotsadm"

	options := identitydeploy.Options{
		NamePrefix:         KotsadmNamePrefix,
		IdentitySpec:       getIdentitySpec(identityConfig.Spec, ingressConfig.Spec),
		IdentityConfigSpec: identityConfig.Spec,
		IsOpenShift:        false, // TODO (ethan): openshift support
		ImageRewriteFn:     imageRewriteKotsadmRegistry(namespace, registryOptions),
		ProxyEnv:           proxyEnv,
		Builder:            nil,
	}

	if err := migrateClientSecret(ctx, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to migrate client secret")
	}

	postgresConfig := kotsv1beta1.IdentityPostgresConfig{
		Host:     "kotsadm-postgres",
		Database: "dex",
		User:     "dex",
	}
	if err := identitydeploy.EnsurePostgresSecret(context.TODO(), clientset, namespace, KotsadmNamePrefix, nil, postgresConfig, nil); err != nil {
		return errors.Wrap(err, "failed to ensure postgres secret")
	}

	if err := identitydeploy.EnsureClientSecret(ctx, clientset, namespace, KotsadmNamePrefix, nil); err != nil {
		return errors.Wrap(err, "failed to ensure client secret")
	}

	return identitydeploy.Deploy(ctx, clientset, namespace, options)
}

func Configure(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig, proxyEnv map[string]string) error {
	identityConfig.Spec.ClientID = "kotsadm"

	options := identitydeploy.Options{
		NamePrefix:         KotsadmNamePrefix,
		IdentitySpec:       getIdentitySpec(identityConfig.Spec, ingressConfig.Spec),
		IdentityConfigSpec: identityConfig.Spec,
		IsOpenShift:        false,
		ImageRewriteFn:     nil,
		ProxyEnv:           proxyEnv,
		Builder:            nil,
	}

	return identitydeploy.Configure(ctx, clientset, namespace, options)
}

func Undeploy(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	return identitydeploy.Undeploy(ctx, clientset, namespace, KotsadmNamePrefix)
}

func imageRewriteKotsadmRegistry(namespace string, registryOptions *kotsadmtypes.KotsadmOptions) identitydeploy.ImageRewriteFunc {
	secret := kotsadmversion.KotsadmPullSecret(namespace, *registryOptions)

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
		image = fmt.Sprintf("%s/%s:%s", kotsadmversion.KotsadmRegistry(*registryOptions), imageName, kotsadmversion.KotsadmTag(*registryOptions))

		if secret != nil {
			imagePullSecrets = []corev1.LocalObjectReference{
				{Name: secret.ObjectMeta.Name},
			}
		}
		return image, imagePullSecrets, err
	}
}

func getIdentitySpec(identityConfigSpec kotsv1beta1.IdentityConfigSpec, ingressConfigSpec kotsv1beta1.IngressConfigSpec) kotsv1beta1.IdentitySpec {
	return kotsv1beta1.IdentitySpec{
		OIDCRedirectURIs: []string{getRedirectURI(identityConfigSpec, ingressConfigSpec)},
	}
}

func getRedirectURI(identityConfigSpec kotsv1beta1.IdentityConfigSpec, ingressConfigSpec kotsv1beta1.IngressConfigSpec) string {
	kotsadmAddress := identityConfigSpec.AdminConsoleAddress
	if kotsadmAddress == "" && ingressConfigSpec.Enabled {
		kotsadmAddress = ingress.GetAddress(ingressConfigSpec)
	}
	return fmt.Sprintf("%s/api/v1/oidc/login/callback", kotsadmAddress)
}

func migrateClientSecret(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	client, err := getKotsadmOIDCClientFromDexConfig(ctx, clientset, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to get existing oidc client from dex config")
	}
	if client == nil || client.Secret == "" {
		return nil
	}

	secret := identitydeploy.ClientSecretResource(KotsadmNamePrefix, client.Secret, nil)
	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	return errors.Wrap(err, "failed to create secret")
}
