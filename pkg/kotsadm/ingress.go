package kotsadm

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/ingress"
	ingresstypes "github.com/replicatedhq/kots/pkg/ingress/types"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func EnsureIngress(ctx context.Context, namespace string, clientset *kubernetes.Clientset, ingressConfig ingresstypes.Config) error {
	if ingressConfig.Ingress == nil {
		return DeleteIngress(ctx, namespace, clientset)
	}
	kotsadmIngress := kotsadmIngress(namespace, *ingressConfig.Ingress)
	return ingress.EnsureIngress(ctx, clientset, namespace, kotsadmIngress)
}

func DeleteIngress(ctx context.Context, namespace string, clientset *kubernetes.Clientset) error {
	err := clientset.ExtensionsV1beta1().Ingresses(namespace).Delete(ctx, "kotsadm", metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete ingress")
}
