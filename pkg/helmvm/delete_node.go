package helmvm

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func DeleteNode(ctx context.Context, client kubernetes.Interface, restconfig *rest.Config, node *corev1.Node) error {
	return nil
}
