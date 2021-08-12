package resources

import (
	"context"

	"github.com/pkg/errors"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func EnsurePrivateKotsadmRegistrySecret(namespace string, kotsadmOptions types.KotsadmOptions, clientset kubernetes.Interface) error {
	_, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), types.PrivateKotsadmRegistrySecret, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing private kotsadm registry secret")
		}

		secret := kotsadmobjects.PrivateKotsadmRegistrySecret(namespace, kotsadmOptions)
		if secret == nil {
			return nil
		}

		_, err := clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create private kotsadm registry secret")
		}
	}

	return nil
}
