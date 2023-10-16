package helmvm

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/replicatedhq/kots/pkg/helmvm/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// GetNode will get a node with stats and podlist
func GetNode(ctx context.Context, client kubernetes.Interface, nodeName string) (*types.Node, error) {
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get node %s: %w", nodeName, err)
	}

	clientConfig, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster config: %w", err)
	}

	metricsClient, err := metricsv.NewForConfig(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics client: %w", err)
	}

	nodePods, err := podsOnNode(ctx, client, nodeName)
	if err != nil {
		return nil, fmt.Errorf("pods per node: %w", err)
	}

	cpuCapacity := types.CapacityAvailable{}
	memoryCapacity := types.CapacityAvailable{}
	podCapacity := types.CapacityAvailable{}

	memoryCapacity.Capacity = float64(node.Status.Capacity.Memory().Value()) / math.Pow(2, 30) // capacity in GB

	cpuCapacity.Capacity, err = strconv.ParseFloat(node.Status.Capacity.Cpu().String(), 64)
	if err != nil {
		return nil, fmt.Errorf("parse CPU capacity %q for node %s: %w", node.Status.Capacity.Cpu().String(), node.Name, err)
	}

	podCapacity.Capacity = float64(node.Status.Capacity.Pods().Value())

	nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, node.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pod metrics: %w", err)
	}

	if nodeMetrics.Usage.Memory() != nil {
		memoryCapacity.Available = memoryCapacity.Capacity - float64(nodeMetrics.Usage.Memory().Value())/math.Pow(2, 30)
	}

	if nodeMetrics.Usage.Cpu() != nil {
		cpuCapacity.Available = cpuCapacity.Capacity - nodeMetrics.Usage.Cpu().AsApproximateFloat64()
	}

	podCapacity.Available = podCapacity.Capacity - float64(len(nodePods))

	nodeLabelArray := []string{}
	for k, v := range node.Labels {
		nodeLabelArray = append(nodeLabelArray, fmt.Sprintf("%s:%s", k, v))
	}

	return &types.Node{
		Name:           node.Name,
		IsConnected:    isConnected(*node),
		IsReady:        isReady(*node),
		IsPrimaryNode:  isPrimary(*node),
		CanDelete:      node.Spec.Unschedulable && !isConnected(*node),
		KubeletVersion: node.Status.NodeInfo.KubeletVersion,
		CPU:            cpuCapacity,
		Memory:         memoryCapacity,
		Pods:           podCapacity,
		Labels:         nodeLabelArray,
		Conditions:     findNodeConditions(node.Status.Conditions),
		PodList:        nodePods,
	}, nil
}

func podsOnNode(ctx context.Context, client kubernetes.Interface, nodeName string) ([]corev1.Pod, error) {
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	toReturn := []corev1.Pod{}

	for _, ns := range namespaces.Items {
		nsPods, err := client.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName)})
		if err != nil {
			return nil, fmt.Errorf("list pods on %s in namespace %s: %w", nodeName, ns.Name, err)
		}

		toReturn = append(toReturn, nsPods.Items...)
	}
	return toReturn, nil
}
