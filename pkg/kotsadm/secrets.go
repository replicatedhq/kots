package kotsadm

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ensureSecrets(namespace string, clientset *kubernetes.Clientset) error {
	if err := ensureJWTSessionSecret(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure jwt session secret")
	}

	if err := ensurePostgresSecret(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure postgres secret")
	}

	return nil
}

func ensureJWTSessionSecret(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Secrets(namespace).Get("kotsadm-session", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing session secret")
		}

		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-session",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"key": []byte(uuid.New().String()),
			},
		}

		_, err := clientset.CoreV1().Secrets(namespace).Create(secret)
		if err != nil {
			return errors.Wrap(err, "failed to create jwt session secret")
		}
	}

	return nil
}

func ensurePostgresSecret(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Secrets(namespace).Get("kotsadm-postgres", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing postgres secret")
		}

		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-postgres",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"uri": []byte(fmt.Sprintf("postgresql://kotsadm:%s@kotsadm-postgres/kotsadm?connect_timeout=10&sslmode=disable", postgresPassword)),
			},
		}

		_, err := clientset.CoreV1().Secrets(namespace).Create(secret)
		if err != nil {
			return errors.Wrap(err, "failed to create postgres secret")
		}
	}

	return nil
}
