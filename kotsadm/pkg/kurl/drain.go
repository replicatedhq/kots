package kurl

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func DrainNode(ctx context.Context, client kubernetes.Interface, node *corev1.Node) error {
	node.Spec.Unschedulable = true
	node, err := client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "cordon node")
	}
	logger.Infof("Node %s successfully cordoned, continuing to pod eviction", node.Name)

	// evict pods on the node
	labelSelectors := []string{
		// Defer draining self and pods that provide cluster services to other pods
		"app notin (rook-ceph-mon,rook-ceph-osd,rook-ceph-operator,kotsadm),k8s-app!=kube-dns",
		// Drain Rook pods
		"app in (rook-ceph-mon,rook-ceph-osd,rook-ceph-operator)",
		// Drain dns pod
		"k8s-app=kube-dns",
		// Drain self
		"app=kotsadm",
	}
	for _, labelSelector := range labelSelectors {
		opts := metav1.ListOptions{
			LabelSelector: labelSelector,
			FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name),
		}
		podList, err := client.CoreV1().Pods("").List(ctx, opts)
		if err != nil {
			return errors.Wrapf(err, "list pods by label %q", labelSelector)
		}
		for _, pod := range podList.Items {
			if !shouldDrain(&pod) {
				continue
			}
			eviction := &policyv1beta1.Eviction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				},
			}
			err := client.PolicyV1beta1().Evictions(pod.Namespace).Evict(ctx, eviction)
			if err != nil {
				return errors.Wrapf(err, "evict pod %s/%s", pod.Namespace, pod.Name)
			}
			logger.Infof("Evicting pod %s/%s from node %s", pod.Namespace, pod.Name, node.Name)
		}
	}

	return nil
}

func shouldDrain(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}

	// completed pods always ok to drain
	if isFinished(pod) {
		return true
	}

	if isMirror(pod) {
		logger.Infof("Skipping drain of mirror pod %s in namespace %s", pod.Name, pod.Namespace)
		return false
	}

	// TODO if orphaned it's ok to delete the pod
	if isDaemonSetPod(pod) {
		logger.Infof("Skipping drain of DaemonSet pod %s in namespace %s", pod.Name, pod.Namespace)
		return false
	}

	return true
}

func isFinished(pod *corev1.Pod) bool {
	if pod.Status.Phase == corev1.PodSucceeded {
		return true
	}
	if pod.Status.Phase == corev1.PodFailed {
		return true
	}

	return false
}

func isMirror(pod *corev1.Pod) bool {
	_, ok := pod.Annotations["kubernetes.io/config.mirror"]

	return ok
}

func isDaemonSetPod(pod *corev1.Pod) bool {
	for _, owner := range pod.OwnerReferences {
		if owner.Controller == nil {
			continue
		}
		if !*owner.Controller {
			continue
		}
		if owner.Kind == "DaemonSet" {
			return true
		}
	}

	return false
}
