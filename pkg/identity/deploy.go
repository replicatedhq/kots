package identity

import (
	"context"
	"fmt"
	"log"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	KotsadmNamePrefix = "kotsadm"
)

func Deploy(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig, registryOptions *kotsadmtypes.KotsadmOptions, proxyEnv map[string]string, applyAppBranding bool) error {
	identityConfig.Spec.ClientID = "kotsadm"

	options := identitydeploy.Options{
		NamePrefix:         KotsadmNamePrefix,
		IdentitySpec:       getIdentitySpec(ctx, clientset, namespace, identityConfig.Spec, ingressConfig.Spec, applyAppBranding),
		IdentityConfigSpec: identityConfig.Spec,
		IsOpenShift:        k8sutil.IsOpenShift(clientset),
		ProxyEnv:           proxyEnv,
		Builder:            nil,
	}

	if !kotsutil.IsKurl(clientset) || namespace != metav1.NamespaceDefault {
		options.ImageRewriteFn = kotsadmversion.ImageRewriteKotsadmRegistry(namespace, registryOptions)
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

func Configure(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig, proxyEnv map[string]string, applyAppBranding bool) error {
	identityConfig.Spec.ClientID = "kotsadm"

	options := identitydeploy.Options{
		NamePrefix:         KotsadmNamePrefix,
		IdentitySpec:       getIdentitySpec(ctx, clientset, namespace, identityConfig.Spec, ingressConfig.Spec, applyAppBranding),
		IdentityConfigSpec: identityConfig.Spec,
		IsOpenShift:        k8sutil.IsOpenShift(clientset),
		ImageRewriteFn:     nil,
		ProxyEnv:           proxyEnv,
		Builder:            nil,
	}

	return identitydeploy.Configure(ctx, clientset, namespace, options)
}

func Undeploy(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	return identitydeploy.Undeploy(ctx, clientset, namespace, KotsadmNamePrefix)
}

func getIdentitySpec(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfigSpec kotsv1beta1.IdentityConfigSpec, ingressConfigSpec kotsv1beta1.IngressConfigSpec, applyAppBranding bool) kotsv1beta1.IdentitySpec {
	// NOTE: when the user adds a second app the branding won't change
	webConfig, err := getWebConfig(ctx, clientset, namespace, applyAppBranding)
	if err != nil {
		log.Printf("Failed to get branding: %v", err)
	}
	return kotsv1beta1.IdentitySpec{
		IdentityIssuerURL:           DexIssuerURL(identityConfigSpec),
		OIDCRedirectURIs:            []string{getRedirectURI(identityConfigSpec, ingressConfigSpec)},
		OAUTH2AlwaysShowLoginScreen: true,
		WebConfig:                   webConfig,
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
