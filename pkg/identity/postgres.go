package identity

import (
	"context"

	"github.com/pkg/errors"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/segmentio/ksuid"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	PostgresSecretName = "kotsadm-dex-postgres"
)

func EnsurePostgresSecret(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	secret := postgresSecretResource(PostgresSecretName)

	_, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing secret")
		}

		_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create secret")
		}

		return nil
	}

	// no patch needed

	return nil
}

func postgresSecretResource(secretName string) *corev1.Secret {
	generatedPassword := ksuid.New().String()

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   secretName,
			Labels: kotsadmtypes.GetKotsadmLabels(identitydeploy.AdditionalLabels("kotsadm")),
		},
		Data: map[string][]byte{
			"password": []byte(generatedPassword),
		},
	}
}
