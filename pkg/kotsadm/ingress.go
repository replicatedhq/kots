package kotsadm

import (
	"context"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/ingress"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func EnsureIngress(ctx context.Context, namespace string, clientset *kubernetes.Clientset, ingressSpec kotsv1beta1.IngressConfigSpec) error {
	if !ingressSpec.Enabled || ingressSpec.Ingress == nil {
		return DeleteIngress(ctx, namespace, clientset)
	}
	kotsadmIngress := kotsadmIngress(namespace, *ingressSpec.Ingress)
	return ingress.EnsureIngress(ctx, clientset, namespace, kotsadmIngress)
}

func DeleteIngress(ctx context.Context, namespace string, clientset *kubernetes.Clientset) error {
	err := clientset.ExtensionsV1beta1().Ingresses(namespace).Delete(ctx, "kotsadm", metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete ingress")
}
