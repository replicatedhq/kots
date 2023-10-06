package helmvm

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func DrainNode(ctx context.Context, client kubernetes.Interface, node *corev1.Node) error {
	return nil
}
