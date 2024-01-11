package util

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ThisImage looks for either a deployment 'kotsadm' or a statefulset 'kotsadm' in the current namespace
// it returns the image of the first container in the pod template
func ThisImage(ctx context.Context, client kubernetes.Interface) (string, error) {
	deploy, err := client.AppsV1().Deployments(PodNamespace).Get(ctx, "kotsadm", metav1.GetOptions{})
	if err == nil {
		return deploy.Spec.Template.Spec.Containers[0].Image, nil
	}

	statefulset, err := client.AppsV1().StatefulSets(PodNamespace).Get(ctx, "kotsadm", metav1.GetOptions{})
	if err == nil {
		return statefulset.Spec.Template.Spec.Containers[0].Image, nil
	}

	return "", fmt.Errorf("failed to find deployment or statefulset")

}
