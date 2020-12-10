package identity

import (
	"context"

	oidc "github.com/coreos/go-oidc"
	dexstorage "github.com/dexidp/dex/storage"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"k8s.io/client-go/kubernetes"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

func GetKotsadmOIDCProvider(ctx context.Context, namespace string) (*oidc.Provider, error) {
	cfg, err := k8sconfig.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubernetes config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client set")
	}

	dexConfig, err := getKotsadmDexConfig(ctx, clientset, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm dex config")
	}

	identityConfig, err := GetConfig(ctx, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get identity config")
	}

	httpClient, err := HTTPClient(ctx, namespace, *identityConfig)
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

func GetKotsadmOAuth2Config(ctx context.Context, namespace string) (*oauth2.Config, error) {
	cfg, err := k8sconfig.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubernetes config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client set")
	}

	dexConfig, err := getKotsadmDexConfig(ctx, clientset, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm dex config")
	}

	provider, err := GetKotsadmOIDCProvider(ctx, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm oidc provider")
	}

	var kotsadmClient *dexstorage.Client
	for _, client := range dexConfig.StaticClients {
		if client.ID == "kotsadm" {
			kotsadmClient = &client
			break
		}
	}
	if kotsadmClient == nil {
		return nil, errors.New("kotsadm dex client not found")
	}

	oauth2Config := oauth2.Config{
		ClientID:     kotsadmClient.ID,
		ClientSecret: kotsadmClient.Secret,
		Endpoint:     provider.Endpoint(),
		Scopes:       getScopes(),
		RedirectURL:  kotsadmClient.RedirectURIs[0],
	}

	return &oauth2Config, nil
}

func getScopes() []string {
	return []string{"openid", "email", "groups"}
}
