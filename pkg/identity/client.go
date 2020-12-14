package identity

import (
	"context"

	oidc "github.com/coreos/go-oidc"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/identity/client"
	"golang.org/x/oauth2"
	"k8s.io/client-go/kubernetes"
)

func GetKotsadmOIDCProvider(ctx context.Context, clientset kubernetes.Interface, namespace string) (*oidc.Provider, error) {
	dexConfig, err := getKotsadmDexConfig(ctx, clientset, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm dex config")
	}
	if dexConfig == nil {
		return nil, errors.Wrap(err, "dex config not found")
	}

	identityConfig, err := GetConfig(ctx, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get identity config")
	}

	httpClient, err := client.HTTPClient(ctx, namespace, *identityConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init http client")
	}

	oidcClientCtx := oidc.ClientContext(ctx, httpClient)
	provider, err := oidc.NewProvider(oidcClientCtx, dexConfig.Issuer)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query provider %q", dexConfig.Issuer)
	}

	return provider, nil
}

func GetKotsadmOAuth2Config(ctx context.Context, clientset kubernetes.Interface, namespace string, provider oidc.Provider) (*oauth2.Config, error) {
	client, err := getOIDCClient(ctx, clientset, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get oidc client secret")
	}
	if client == nil {
		return nil, errors.Wrap(err, "oidc client not found")
	}

	oauth2Config := oauth2.Config{
		ClientID:     client.ID,
		ClientSecret: client.Secret,
		Endpoint:     provider.Endpoint(),
		Scopes:       getScopes(),
		RedirectURL:  client.RedirectURIs[0],
	}

	return &oauth2Config, nil
}

func getScopes() []string {
	return []string{"openid", "email", "groups"}
}
