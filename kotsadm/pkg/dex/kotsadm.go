package dex

import (
	"context"
	"sync"
	"time"

	oidc "github.com/coreos/go-oidc"
	dexstorage "github.com/dexidp/dex/storage"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/k8s"
	"github.com/replicatedhq/kots/pkg/identity"
	dextypes "github.com/replicatedhq/kots/pkg/identity/types/dex"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DexStateSecretName = "kotsadm-dex-state"
)

var stateMtx sync.Mutex

func GetKotsadmDexConfig(ctx context.Context, namespace string) (*dextypes.Config, error) {
	clientset, err := k8s.Clientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s client set")
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, identity.DexSecretName, metav1.GetOptions{})
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

func GetKotsadmOIDCProvider(ctx context.Context, namespace string) (*oidc.Provider, error) {
	dexConfig, err := GetKotsadmDexConfig(ctx, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm dex config")
	}

	identityConfig, err := identity.GetConfig(ctx, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get identity config")
	}

	httpClient, err := identity.HTTPClient(ctx, namespace, *identityConfig)
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
	dexConfig, err := GetKotsadmDexConfig(ctx, namespace)
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
	// TODO "offline_access"
	return []string{"openid", "profile", "email", "groups"}
}

func SetDexState(ctx context.Context, namespace string, state string) error {
	stateMtx.Lock()
	defer stateMtx.Unlock()

	secret := stateSecretResource(DexStateSecretName, state)

	clientset, err := k8s.Clientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, DexStateSecretName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing dex state secret")
		}

		_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create dex state secret")
		}

		return nil
	}

	existingSecret = updateStateSecret(existingSecret, secret)

	_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update dex state secret")
	}

	return nil
}

func GetDexState(ctx context.Context, namespace string, state string) (string, error) {
	clientset, err := k8s.Clientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to get k8s client set")
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, DexStateSecretName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get dex state secret")
	}

	return string(secret.Data[state]), nil
}

func ResetDexState(ctx context.Context, namespace string, state string) error {
	stateMtx.Lock()
	defer stateMtx.Unlock()

	clientset, err := k8s.Clientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, DexStateSecretName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get dex state secret")
	}

	delete(secret.Data, state)

	_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
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
			Labels: kotsadmtypes.GetKotsadmLabels(identity.AdditionalLabels),
		},
		Data: map[string][]byte{
			state: []byte(time.Now().UTC().Format(time.RFC3339)),
		},
	}
}

func updateStateSecret(existingSecret, desiredSecret *corev1.Secret) *corev1.Secret {
	existingSecret.Data = mergeMaps(existingSecret.Data, desiredSecret.Data)
	expireOldDexStates(existingSecret)
	return existingSecret
}

func expireOldDexStates(secret *corev1.Secret) error {
	for s, t := range secret.Data {
		stateTime, err := time.Parse(time.RFC3339, string(t))
		if err != nil {
			return errors.Wrap(err, "failed to parse state time")
		}

		ttlTime := time.Now().Add(-24 * time.Hour)
		if stateTime.Before(ttlTime) {
			delete(secret.Data, s)
			continue
		}
	}

	return nil
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
