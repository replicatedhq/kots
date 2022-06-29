package image

import (
	"context"
	"fmt"
	"strings"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// MinioImage looks through the nodes in the cluster and finds nodes that have already pulled Minio, and then finds the latest image tag listed
func GetMinioImage(clientset kubernetes.Interface, kotsadmNamespace string) (string, error) {
	/*
	 *  If it is a kurl instance with Minio add-on, use the same image that's used by the add-on.
	 *  If it is not a kurl instance, return the static image name present in the bundle.
	 */
	if !kotsutil.IsKurl(clientset) || kotsadmNamespace != metav1.NamespaceDefault {
		return Minio, nil
	}

	deployment, err := clientset.AppsV1().Deployments("minio").Get(context.TODO(), "minio", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get minio deployment: %w", err)
	}

	for _, container := range deployment.Spec.Template.Spec.Containers {
		if strings.Contains(container.Image, "minio/minio:RELEASE.") {
			return container.Image, nil
		}
	}

	return "", nil
}
