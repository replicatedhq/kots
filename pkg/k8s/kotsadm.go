package k8s

import (
	"context"
	"os"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func FindKotsadmImage(namespace string) (string, error) {
	client, err := Clientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to get k8s client set")
	}

	if os.Getenv("KOTSADM_ENV") == "dev" {
		namespace = os.Getenv("POD_NAMESPACE")
	}

	kotsadmDeployment, err := client.AppsV1().Deployments(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get kotsadm deployment")
	}

	apiContainerIndex := -1
	for i, container := range kotsadmDeployment.Spec.Template.Spec.Containers {
		if container.Name == "kotsadm" {
			apiContainerIndex = i
			break
		}
	}

	if apiContainerIndex == -1 {
		return "", errors.New("kotsadm container not found")
	}

	kotsadmImage := kotsadmDeployment.Spec.Template.Spec.Containers[apiContainerIndex].Image

	return kotsadmImage, nil
}

// IsKotsadmClusterScoped will check if kotsadm has cluster scope access or not
func IsKotsadmClusterScoped(ctx context.Context, clientset kubernetes.Interface, namespace string) bool {
	rb, err := clientset.RbacV1().ClusterRoleBindings().Get(ctx, "kotsadm-rolebinding", metav1.GetOptions{})
	if err != nil {
		return false
	}
	for _, s := range rb.Subjects {
		if s.Kind != "ServiceAccount" {
			continue
		}
		if s.Name != "kotsadm" {
			continue
		}
		if s.Namespace != "" && s.Namespace == namespace {
			return true
		}
		if s.Namespace == "" && namespace == metav1.NamespaceDefault {
			return true
		}
	}
	return false
}
