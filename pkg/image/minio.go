package image

import (
	"context"
	"fmt"
	"strings"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// MinioImage looks through the nodes in the cluster and finds nodes that have already pulled Minio, and then finds the latest image tag listed
func GetMinioImage(clientset kubernetes.Interface, kotsadmNamespace string) (string, error) {
	/*
	 *  In existing install with limited RBAC, kotsadm does not have previliges to run Nodes() API.
	 *  If it is a kurl instance, then use search logic to find the best minio image.
	 *  If it is not a kurl instance, return the static image name present in the bundle.
	 */
	if !kotsutil.IsKurl(clientset) || kotsadmNamespace != metav1.NamespaceDefault {
		return Minio, nil
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list nodes with minio image: %w", err)
	}

	bestMinioImage := ""
	for _, node := range nodes.Items {
		for _, image := range node.Status.Images {
			for _, name := range image.Names {
				if strings.Contains(name, "minio/minio:RELEASE.") {
					// this is a minio image!
					if bestMinioImage == "" {
						bestMinioImage = name
					} else {
						bestMinioImage = latestMinioImage(bestMinioImage, name)
					}
				}
			}
		}
	}

	return bestMinioImage, nil
}

// latestMinioImage returns the later of two provided images
func latestMinioImage(a, b string) string {
	// first extract the tags to compare
	splita := strings.Split(a, ":")
	taga := splita[len(splita)-1]

	splitb := strings.Split(b, ":")
	tagb := splitb[len(splitb)-1]

	if strings.Compare(taga, tagb) > 0 {
		return a
	}
	return b
}
