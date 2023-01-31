package k8sutil

import (
	"context"

	"github.com/pkg/errors"
	kotstypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func AddLabelsToSecret(client kubernetes.Interface, namespace string, secretName string, labels map[string]string) (*corev1.Secret, error) {
	existingSecret, err := client.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to read secret")
	} else if kuberneteserrors.IsNotFound(err) {
		return nil, nil
	}

	existingSecret.ObjectMeta.Labels = kotstypes.MergeLabels(existingSecret.ObjectMeta.Labels, labels)
	updatedSecret, err := client.CoreV1().Secrets(namespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to update secret")
	}
	return updatedSecret, nil
}
