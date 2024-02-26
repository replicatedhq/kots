package identity

import (
	"context"
	"strings"

	ghodssyaml "github.com/ghodss/yaml"
	"github.com/pkg/errors"
	dextypes "github.com/replicatedhq/kots/pkg/dex/types"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func getOIDCClient(ctx context.Context, clientset kubernetes.Interface, namespace string) (*dextypes.StorageClient, error) {
	client, err := getKotsadmOIDCClientFromDexConfig(ctx, clientset, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get existing oidc client from dex config")
	}

	if client == nil {
		return nil, nil
	}

	if client.Secret != "" {
		return client, nil
	}

	clientSecret, err := identitydeploy.GetClientSecret(ctx, clientset, namespace, KotsadmNamePrefix)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get existing dex config")
	}
	client.Secret = clientSecret

	return client, nil
}

func getKotsadmOIDCClientFromDexConfig(ctx context.Context, clientset kubernetes.Interface, namespace string) (*dextypes.StorageClient, error) {
	existingConfig, err := getKotsadmDexConfig(ctx, clientset, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get existing dex config")
	}

	if existingConfig == nil {
		return nil, nil
	}

	for _, client := range existingConfig.StaticClients {
		if client.ID == "kotsadm" && !strings.HasPrefix(client.Secret, "$") {
			return &client, nil
		}
	}

	return nil, errors.New("oidc client not found")
}

func getKotsadmDexConfig(ctx context.Context, clientset kubernetes.Interface, namespace string) (*dextypes.Config, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, "kotsadm-dex", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get kotsadm-dex secret")
	}

	marshalledConfig := secret.Data["dexConfig.yaml"]
	config := dextypes.Config{}
	if err := ghodssyaml.Unmarshal(marshalledConfig, &config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal kotsadm dex config")
	}

	return &config, nil
}
