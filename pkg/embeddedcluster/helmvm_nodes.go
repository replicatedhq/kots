package embeddedcluster

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/embeddedcluster/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// GetNodes will get a list of nodes with stats
func GetNodes(ctx context.Context, client kubernetes.Interface) (*types.EmbeddedClusterNodes, error) {
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list nodes")
	}

	clientConfig, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	metricsClient, err := metricsv.NewForConfig(clientConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create metrics client")
	}

	toReturn := types.EmbeddedClusterNodes{
		IsEmbeddedClusterEnabled: util.IsEmbeddedCluster(),
	}

	for _, node := range nodes.Items {
		nodeMet, err := nodeMetrics(ctx, client, metricsClient, node)
		if err != nil {
			return nil, errors.Wrap(err, "node metrics")
		}

		toReturn.Nodes = append(toReturn.Nodes, *nodeMet)
	}

	isHA, err := IsHA(client)
	if err != nil {
		return nil, errors.Wrap(err, "is ha")
	}
	toReturn.HA = isHA

	return &toReturn, nil
}

func findNodeConditions(conditions []corev1.NodeCondition) types.NodeConditions {
	discoveredConditions := types.NodeConditions{}
	for _, condition := range conditions {
		if condition.Type == "MemoryPressure" {
			discoveredConditions.MemoryPressure = condition.Status == corev1.ConditionTrue
		}
		if condition.Type == "DiskPressure" {
			discoveredConditions.DiskPressure = condition.Status == corev1.ConditionTrue
		}
		if condition.Type == "PIDPressure" {
			discoveredConditions.PidPressure = condition.Status == corev1.ConditionTrue
		}
		if condition.Type == "Ready" {
			discoveredConditions.Ready = condition.Status == corev1.ConditionTrue
		}
	}
	return discoveredConditions
}

func isConnected(node corev1.Node) bool {
	for _, taint := range node.Spec.Taints {
		if taint.Key == "node.kubernetes.io/unreachable" {
			return false
		}
	}

	return true
}

func isReady(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == "Ready" {
			return condition.Status == corev1.ConditionTrue
		}
	}

	return false
}

func isPrimary(node corev1.Node) bool {
	for label := range node.ObjectMeta.Labels {
		if label == "node-role.kubernetes.io/master" {
			return true
		}
		if label == "node-role.kubernetes.io/control-plane" {
			return true
		}
	}

	return false
}
