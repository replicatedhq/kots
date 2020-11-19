package dex

import (
	"context"
	"net/http"
	"os"

	oidc "github.com/coreos/go-oidc"
	dexstorage "github.com/dexidp/dex/storage"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/k8s"
	"github.com/replicatedhq/kots/pkg/identity"
	dextypes "github.com/replicatedhq/kots/pkg/identity/types/dex"
	"golang.org/x/oauth2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetKotsadmDexConfig() (*dextypes.Config, error) {
	clientset, err := k8s.Clientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s client set")
	}

	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "test" { // TODO
		namespace = "default"
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), identity.DexSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm-dex secret")
	}

	marshalledConfig := secret.Data["dexConfig.yaml"]
	config := dextypes.Config{}
	if err := yaml.Unmarshal(marshalledConfig, &config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal kotsadm dex config")
	}

	return &config, nil
}

func GetKotsadmOIDCProvider() (*oidc.Provider, error) {
	dexConfig, err := GetKotsadmDexConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm dex config")
	}

	ctx := oidc.ClientContext(context.Background(), http.DefaultClient)
	provider, err := oidc.NewProvider(ctx, dexConfig.Issuer)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query provider %q", dexConfig.Issuer)
	}

	return provider, nil
}

func GetKotsadmOAuth2Config() (*oauth2.Config, error) {
	dexConfig, err := GetKotsadmDexConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm dex config")
	}

	provider, err := GetKotsadmOIDCProvider()
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
	// TODO "offline_access"
	return []string{"openid", "profile", "email", "groups"}
}
