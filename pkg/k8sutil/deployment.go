package k8sutil

import (
	"context"
	"time"

	"github.com/pkg/errors"
	types "github.com/replicatedhq/kots/pkg/k8sutil/types"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
)

func WaitForDeploymentReady(ctx context.Context, clientset kubernetes.Interface, namespace string, deploymentName string, timeout time.Duration) error {
	start := time.Now()

	for {
		d, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
		if err != nil {
			if !kuberneteserrors.IsNotFound(err) {
				return errors.Wrap(err, "failed to get existing deployment")
			}
			return nil
		}

		if d.Status.ObservedGeneration == d.ObjectMeta.Generation && d.Status.ReadyReplicas == *d.Spec.Replicas && d.Status.UnavailableReplicas == 0 {
			return nil
		}

		time.Sleep(time.Second)

		if time.Since(start) > timeout {
			return &types.ErrorTimeout{Message: "timeout waiting for deployment to become ready"}
		}
	}
}

func ScaleDownDeployment(ctx context.Context, clientset kubernetes.Interface, namespace string, deploymentName string) error {
	d, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}
		return nil
	}

	d.Spec.Replicas = pointer.Int32Ptr(0)

	_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, d, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update deployment")
	}

	return nil
}
