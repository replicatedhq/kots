package ingress

import (
	"context"

	"github.com/pkg/errors"
	networkingv1 "k8s.io/api/networking/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func EnsureIngress(ctx context.Context, clientset kubernetes.Interface, namespace string, ingress *networkingv1.Ingress) error {
	existing, err := clientset.NetworkingV1().Ingresses(namespace).Get(ctx, ingress.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing ingress")
		}

		_, err = clientset.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create ingress")
		}

		return nil
	}

	existing = updateIngress(existing, ingress)

	_, err = clientset.NetworkingV1().Ingresses(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update ingress")
	}

	return nil
}

func updateIngress(existing, desiredIngress *networkingv1.Ingress) *networkingv1.Ingress {
	existing.Annotations = desiredIngress.Annotations
	existing.Spec.Rules = desiredIngress.Spec.Rules
	existing.Spec.TLS = desiredIngress.Spec.TLS

	return existing
}
