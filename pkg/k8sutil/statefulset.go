package k8sutil

import (
	"context"
	"time"

	"github.com/pkg/errors"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func WaitForStatefulSetReady(ctx context.Context, clientset kubernetes.Interface, namespace string, statefulSetName string, timeout time.Duration) error {
	start := time.Now()

	for {
		s, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
		if err != nil {
			if !kuberneteserrors.IsNotFound(err) {
				return errors.Wrap(err, "failed to get existing statefulset")
			}
			return nil
		}

		if s.Status.ObservedGeneration == s.ObjectMeta.Generation && s.Status.ReadyReplicas == *s.Spec.Replicas {
			return nil
		}

		time.Sleep(time.Second)

		if time.Now().Sub(start) > timeout {
			return &kotsadmtypes.ErrorTimeout{Message: "timeout waiting for statefulset to become ready"}
		}
	}
}
