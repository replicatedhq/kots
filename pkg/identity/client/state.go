package client

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const OIDCStateSecretName = "kotsadm-dex-state"

var stateMtx sync.Mutex

func SetOIDCState(ctx context.Context, namespace string, state string) error {
	stateMtx.Lock()
	defer stateMtx.Unlock()

	secret := stateSecretResource(OIDCStateSecretName, state)

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing oidc state secret")
		}

		_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create oidc state secret")
		}

		return nil
	}

	existingSecret = updateStateSecret(existingSecret, secret)

	_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update oidc state secret")
	}

	return nil
}

func GetOIDCState(ctx context.Context, namespace string, state string) (string, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to get k8s client set")
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, OIDCStateSecretName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get oidc state secret")
	}

	return string(secret.Data[state]), nil
}

func ResetOIDCState(ctx context.Context, namespace string, state string) error {
	stateMtx.Lock()
	defer stateMtx.Unlock()

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, OIDCStateSecretName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get oidc state secret")
	}

	delete(secret.Data, state)

	_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update oidc state secret")
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
			Labels: kotsadmtypes.GetKotsadmLabels(identitydeploy.AdditionalLabels("kotsadm", nil)),
		},
		Data: map[string][]byte{
			state: []byte(time.Now().UTC().Format(time.RFC3339)),
		},
	}
}

func updateStateSecret(existingSecret, desiredSecret *corev1.Secret) *corev1.Secret {
	existingSecret.Data = mergeMaps(existingSecret.Data, desiredSecret.Data)
	expireOldOIDCStates(existingSecret)
	return existingSecret
}

func expireOldOIDCStates(secret *corev1.Secret) error {
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
