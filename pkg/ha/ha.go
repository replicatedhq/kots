package ha

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
)

const (
	KUBERNETES_HA_MIN_NODE_COUNT = 3
	RQLITE_HA_NODE_COUNT         = 3
	REASON_NOT_ENOUGH_NODES      = "not enough nodes to run in HA mode"
)

func CanRunHA(ctx context.Context, clientset kubernetes.Interface) (bool, string, error) {
	labelSelector, err := kotsadmobjects.DefaultKOTSNodeLabelSelector()
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get node label selector")
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return false, "", errors.Wrap(err, "failed to list nodes")
	}

	if len(nodes.Items) < KUBERNETES_HA_MIN_NODE_COUNT {
		return false, REASON_NOT_ENOUGH_NODES, nil
	}

	return true, "", nil
}

func EnableHA(ctx context.Context, clientset kubernetes.Interface, namespace string, timeout time.Duration) (finalErr error) {
	if err := scaleKotsadmRqlite(ctx, clientset, namespace, RQLITE_HA_NODE_COUNT); err != nil {
		return errors.Wrap(err, "failed to scale up kotsadm-rqlite")
	}

	if err := k8sutil.WaitForStatefulSetReady(ctx, clientset, namespace, "kotsadm-rqlite", timeout); err != nil {
		return errors.Wrap(err, "failed to wait for kotsadm-rqlite to be ready")
	}

	return nil
}

func scaleKotsadmRqlite(ctx context.Context, clientset kubernetes.Interface, namespace string, replicas int32) error {
	rqliteSts, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, "kotsadm-rqlite", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get kotsadm-rqlite statefulset")
	}

	rqliteSts.Spec.Replicas = pointer.Int32Ptr(replicas)

	for i, arg := range rqliteSts.Spec.Template.Spec.Containers[0].Args {
		if strings.HasPrefix(arg, "-bootstrap-expect") {
			rqliteSts.Spec.Template.Spec.Containers[0].Args[i] = fmt.Sprintf("-bootstrap-expect=%d", replicas)
			break
		}
	}

	if _, err := clientset.AppsV1().StatefulSets(namespace).Update(ctx, rqliteSts, metav1.UpdateOptions{}); err != nil {
		return errors.Wrap(err, "failed to update kotsadm-rqlite statefulset")
	}

	return nil
}
