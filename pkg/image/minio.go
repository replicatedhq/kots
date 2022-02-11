package image

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// MinioImage looks through the nodes in the cluster and finds nodes that have already pulled Minio, and then finds the latest image tag listed
func MinioImage(clientset kubernetes.Interface) (string, error) {
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
