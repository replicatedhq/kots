package image

import (
	"context"
	"regexp"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kurl"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// minioReleaseImageRegexp matches any repository whose image name is "minio" with an
// upstream RELEASE tag, e.g. "minio/minio:RELEASE.2025-10-15T17-29-55Z" and
// "kurlsh/minio:RELEASE.2025-10-15T17-29-55Z".
var minioReleaseImageRegexp = regexp.MustCompile(`(^|/)minio:RELEASE\.`)

// MinioImage looks through the nodes in the cluster and finds nodes that have already pulled Minio, and then finds the latest image tag listed
func GetMinioImage(clientset kubernetes.Interface, kotsadmNamespace string) (string, error) {
	/*
	 *  If it is a kurl instance with Minio add-on, use the same image that's used by the add-on.
	 *  If it is not a kurl instance, return the static image name present in the bundle.
	 */

	// expected to fail for minimal rbac
	isKurl, _ := kurl.IsKurl(clientset)
	if !isKurl || kotsadmNamespace != metav1.NamespaceDefault {
		return Minio, nil
	}

	deployment, err := clientset.AppsV1().Deployments("minio").Get(context.TODO(), "minio", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return "", errors.Wrap(err, "failed to get minio deployment")
	}
	if err == nil {
		for _, container := range deployment.Spec.Template.Spec.Containers {
			if isMinioReleaseImage(container.Image) {
				return container.Image, nil
			}
		}
	}

	// minio deployment doesn't exist, check if ha-minio statefulset exists

	statefulset, err := clientset.AppsV1().StatefulSets("minio").Get(context.TODO(), "ha-minio", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return "", errors.Wrap(err, "failed to get ha-minio statefulset")
	}
	if err == nil {
		for _, container := range statefulset.Spec.Template.Spec.Containers {
			if isMinioReleaseImage(container.Image) {
				return container.Image, nil
			}
		}
	}

	return "", nil
}

// isMinioReleaseImage returns true if the image is a minio image with an upstream
// RELEASE tag, regardless of which repository it was published to. The add-on has
// shipped from more than one repository (minio/minio, kurlsh/minio), so only the
// image name and tag are matched, not the registry or organization.
func isMinioReleaseImage(image string) bool {
	return minioReleaseImageRegexp.MatchString(image)
}
