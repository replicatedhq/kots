package kotsadm

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/ingress"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func EnsureIngress(ctx context.Context, namespace string, clientset kubernetes.Interface, ingressSpec kotsv1beta1.IngressConfigSpec) error {
	if !ingressSpec.Enabled || ingressSpec.Ingress == nil {
		return DeleteIngress(ctx, namespace, clientset)
	}
	kotsadmIngress := kotsadmobjects.KotsadmIngress(namespace, *ingressSpec.Ingress)
	return ingress.EnsureIngress(ctx, clientset, namespace, kotsadmIngress)
}

func DeleteIngress(ctx context.Context, namespace string, clientset kubernetes.Interface) error {
	err := clientset.NetworkingV1().Ingresses(namespace).Delete(ctx, "kotsadm", metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err = nil
	}
	return errors.Wrap(err, "failed to delete ingress")
}
