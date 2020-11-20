package dex

import (
	"context"
	"net/http"
	"os"

	oidc "github.com/coreos/go-oidc"
	corev1 "k8s.io/api/core/v1"
	dexstorage "github.com/dexidp/dex/storage"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/k8s"
	"github.com/replicatedhq/kots/pkg/identity"
	dextypes "github.com/replicatedhq/kots/pkg/identity/types/dex"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"golang.org/x/oauth2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	DexStateSecretName = "kotsadm-dex-state"
)

func GetKotsadmDexConfig() (*dextypes.Config, error) {
	clientset, err := k8s.Clientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s client set")
	}

	secret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), identity.DexSecretName, metav1.GetOptions{})
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

func SetDexState(state string) error {
	secret := stateSecretResource(DexStateSecretName, state)

	clientset, err := k8s.Clientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	existingSecret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), DexStateSecretName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing dex state secret")
		}

		_, err = clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create dex state secret")
		}

		return nil
	}

	existingSecret = updateStateSecret(existingSecret, secret)

	_, err = clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Update(context.TODO(), existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update dex state secret")
	}

	return nil
}

func GetDexState(state string) (string, error) {
	clientset, err := k8s.Clientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to get k8s client set")
	}

	secret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), DexStateSecretName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get dex state secret")
	}

	return string(secret.Data[state]), nil
}

func ResetDexState(state string) error {
	clientset, err := k8s.Clientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	secret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), DexStateSecretName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get dex state secret")
	}

	delete(secret.Data, state)

	_, err = clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Update(context.TODO(), secret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update dex state secret")
	}

	return nil
}

func stateSecretResource(secretName, state string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   secretName,
			Labels: kotsadmtypes.GetKotsadmLabels(identity.DexAdditionalLabels),
		},
		Data: map[string][]byte{
			state: []byte(state),
		},
	}
}

func updateStateSecret(existingSecret, desiredSecret *corev1.Secret) *corev1.Secret {
	existingSecret.Data = mergeMaps(existingSecret.Data, desiredSecret.Data)
	return existingSecret
}

func mergeMaps(existing map[string][]byte, new map[string][]byte) map[string][]byte {
	merged := existing
	if merged == nil {
		merged = make(map[string][]byte)
	}
	for key, value := range new {
		merged[key] = value
	}
	return merged
}