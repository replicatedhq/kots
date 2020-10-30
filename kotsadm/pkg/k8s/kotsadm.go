package k8s

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func FindKotsadmImage(namespace string) (string, error) {
	client, err := Clientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to get k8s client set")
	}

	namespace = "default" // TODOOOOO: REMOVE THIS

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
