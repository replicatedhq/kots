package identity

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	"github.com/replicatedhq/kots/pkg/ingress"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	KotsadmNamePrefix = "kotsadm"
)

func Deploy(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig, registryOptions *kotsadmtypes.KotsadmOptions) error {
	identityConfig.Spec.ClientID = "kotsadm"

	options := identitydeploy.Options{
		NamePrefix:         KotsadmNamePrefix,
		IdentitySpec:       getIdentitySpec(identityConfig.Spec, ingressConfig.Spec),
		IdentityConfigSpec: identityConfig.Spec,
		ImageRewriteFn:     imageRewriteKotsadmRegistry(namespace, registryOptions),
	}

	if err := migrateClientSecret(ctx, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to migrate client secret")
	}

	postgresConfig := kotsv1beta1.IdentityPostgresConfig{
		Host:     "kotsadm-postgres",
		Database: "dex",
		User:     "dex",
	}
	if err := identitydeploy.EnsurePostgresSecret(context.TODO(), clientset, namespace, KotsadmNamePrefix, postgresConfig); err != nil {
		return errors.Wrap(err, "failed to ensure postgres secret")
	}

	if err := identitydeploy.EnsureClientSecret(ctx, clientset, namespace, KotsadmNamePrefix); err != nil {
		return errors.Wrap(err, "failed to ensure client secret")
	}

	return identitydeploy.Deploy(ctx, clientset, namespace, options)
}

func Configure(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig) error {
	identityConfig.Spec.ClientID = "kotsadm"

	options := identitydeploy.Options{
		NamePrefix:         KotsadmNamePrefix,
		IdentitySpec:       getIdentitySpec(identityConfig.Spec, ingressConfig.Spec),
		IdentityConfigSpec: identityConfig.Spec,
	}

	return identitydeploy.Configure(ctx, clientset, namespace, options)
}

func Render(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig, registryOptions *kotsadmtypes.KotsadmOptions) (map[string][]byte, error) {
	identityConfig.Spec.ClientID = "kotsadm"

	options := identitydeploy.Options{
		NamePrefix:         KotsadmNamePrefix,
		IdentitySpec:       getIdentitySpec(identityConfig.Spec, ingressConfig.Spec),
		IdentityConfigSpec: identityConfig.Spec,
		ImageRewriteFn:     imageRewriteKotsadmRegistry(namespace, registryOptions),
	}

	resources, err := identitydeploy.Render(ctx, options)
	if err != nil {
		return nil, err
	}

	postgresSecret, err := renderPostgresSecret(ctx, clientset, namespace)
	if err != nil {
		return nil, err
	}
	resources["postgressecret.yaml"] = postgresSecret

	clientSecret, err := renderClientSecret(ctx, clientset, namespace)
	if err != nil {
		return nil, err
	}
	resources["clientsecret.yaml"] = clientSecret

	return resources, nil
}

func Undeploy(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	return identitydeploy.Undeploy(ctx, clientset, namespace, KotsadmNamePrefix)
}

func imageRewriteKotsadmRegistry(namespace string, registryOptions *kotsadmtypes.KotsadmOptions) identitydeploy.ImageRewriteFunc {
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

func renderPostgresSecret(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]byte, error) {
	postgresSecret, err := identitydeploy.GetPostgresSecret(ctx, clientset, namespace, KotsadmNamePrefix)
	if err != nil && !kuberneteserrors.IsNotFound(errors.Cause(err)) {
		return nil, errors.Wrap(err, "failed to get postgres secret")
	}

	postgresConfig := kotsv1beta1.IdentityPostgresConfig{
		Host:     "kotsadm-postgres",
		Database: "dex",
		User:     "dex",
	}
	if postgresSecret != nil {
		var password []byte
		if len(postgresSecret.Data["password"]) > 0 { // migrate to PGPASSWORD
			password = postgresSecret.Data["password"]
		} else {
			password = postgresSecret.Data["PGPASSWORD"]
		}
		if len(password) > 0 {
			p, err := base64.StdEncoding.DecodeString(string(password))
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode postgres password")
			}
			postgresConfig.Password = string(p)
		}
	}
	resource, err := identitydeploy.RenderPostgresSecret(ctx, KotsadmNamePrefix, postgresConfig)
	return resource, errors.Wrap(err, "failed to render postgres secret")
}

func renderClientSecret(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]byte, error) {
	clientSecret, err := identitydeploy.GetClientSecret(ctx, clientset, namespace, KotsadmNamePrefix)
	if err != nil && !kuberneteserrors.IsNotFound(errors.Cause(err)) {
		return nil, errors.Wrap(err, "failed to get client secret")
	}

	resource, err := identitydeploy.RenderClientSecret(ctx, KotsadmNamePrefix, clientSecret)
	return resource, errors.Wrap(err, "failed to render client secret")
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

	secret := identitydeploy.ClientSecretResource(KotsadmNamePrefix, client.Secret)
	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	return errors.Wrap(err, "failed to create secret")
}
