package kotsadm

import (
	"context"

	"github.com/pkg/errors"
	ingresstypes "github.com/replicatedhq/kots/pkg/ingress/types"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func EnsureIngress(ctx context.Context, namespace string, clientset *kubernetes.Clientset, ingressConfig ingresstypes.Config) error {
	existingIngress, err := clientset.ExtensionsV1beta1().Ingresses(namespace).Get(ctx, "kotsadm", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing kotsadm ingress")
		}

		_, err = clientset.ExtensionsV1beta1().Ingresses(namespace).Create(ctx, kotsadmIngress(namespace, ingressConfig), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create kotsadm ingress")
		}

		return nil
	}

	existingIngress = updateIngress(existingIngress, namespace, ingressConfig)

	_, err = clientset.ExtensionsV1beta1().Ingresses(namespace).Update(ctx, existingIngress, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update kotsadm ingress")
	}

	return nil
}

func DeleteIngress(ctx context.Context, namespace string, clientset *kubernetes.Clientset) error {
	err := clientset.ExtensionsV1beta1().Ingresses(namespace).Delete(ctx, "kotsadm", metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete ingress")
}
