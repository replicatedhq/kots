package embeddedcluster

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/replicatedhq/kots/pkg/embeddedcluster/types"
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

	return nodeMetrics(ctx, client, metricsClient, *node)
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

// nodeMetrics takes a corev1.Node and gets metrics + status for that node
func nodeMetrics(ctx context.Context, client kubernetes.Interface, metricsClient *metricsv.Clientset, node corev1.Node) (*types.Node, error) {
	nodePods, err := podsOnNode(ctx, client, node.Name)
	if err != nil {
		return nil, fmt.Errorf("pods per node: %w", err)
	}

	cpuCapacity := types.CapacityUsed{}
	memoryCapacity := types.CapacityUsed{}
	podCapacity := types.CapacityUsed{}

	memoryCapacity.Capacity = float64(node.Status.Capacity.Memory().Value()) / math.Pow(2, 30) // capacity in GB

	cpuCapacity.Capacity, err = strconv.ParseFloat(node.Status.Capacity.Cpu().String(), 64)
	if err != nil {
		return nil, fmt.Errorf("parse CPU capacity %q for node %s: %w", node.Status.Capacity.Cpu().String(), node.Name, err)
	}

	podCapacity.Capacity = float64(node.Status.Capacity.Pods().Value())

	nodeUsageMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, node.Name, metav1.GetOptions{})
	if err == nil {
		if nodeUsageMetrics.Usage.Memory() != nil {
			memoryCapacity.Used = float64(nodeUsageMetrics.Usage.Memory().Value()) / math.Pow(2, 30)
		}

		if nodeUsageMetrics.Usage.Cpu() != nil {
			cpuCapacity.Used = nodeUsageMetrics.Usage.Cpu().AsApproximateFloat64()
		}
	} else {
		// if we can't get metrics, we'll do nothing for now
		// in the future we may decide to retry or log a warning
	}

	podCapacity.Used = float64(len(nodePods))

	podInfo := []types.PodInfo{}

	for _, pod := range nodePods {
		newInfo := types.PodInfo{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
		}

		podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if err == nil {
			podTotalMemory := 0.0
			podTotalCPU := 0.0
			for _, container := range podMetrics.Containers {
				if container.Usage.Memory() != nil {
					podTotalMemory += float64(container.Usage.Memory().Value()) / math.Pow(2, 20)
				}
				if container.Usage.Cpu() != nil {
					podTotalCPU += container.Usage.Cpu().AsApproximateFloat64() * 1000
				}
			}
			newInfo.Memory = fmt.Sprintf("%.1f MB", podTotalMemory)
			newInfo.CPU = fmt.Sprintf("%.1f m", podTotalCPU)
		}

		podInfo = append(podInfo, newInfo)
	}

	return &types.Node{
		Name:             node.Name,
		IsConnected:      isConnected(node),
		IsReady:          isReady(node),
		IsPrimaryNode:    isPrimary(node),
		CanDelete:        node.Spec.Unschedulable && !isConnected(node),
		KubeletVersion:   node.Status.NodeInfo.KubeletVersion,
		KubeProxyVersion: node.Status.NodeInfo.KubeProxyVersion,
		OperatingSystem:  node.Status.NodeInfo.OperatingSystem,
		KernelVersion:    node.Status.NodeInfo.KernelVersion,
		CPU:              cpuCapacity,
		Memory:           memoryCapacity,
		Pods:             podCapacity,
		Labels:           nodeRolesFromLabels(node.Labels),
		Conditions:       findNodeConditions(node.Status.Conditions),
		PodList:          podInfo,
	}, nil
}

// nodeRolesFromLabels parses a map of k8s node labels, and returns the roles of the node
func nodeRolesFromLabels(labels map[string]string) []string {
	toReturn := []string{}

	numRolesStr, ok := labels[types.EMBEDDED_CLUSTER_ROLE_LABEL]
	if !ok {
		// the first node will not initially have a role label, but is a 'controller'
		if val, ok := labels["node-role.kubernetes.io/control-plane"]; ok && val == "true" {
			return []string{"controller"}
		}
		return nil
	}
	numRoles, err := strconv.Atoi(strings.TrimPrefix(numRolesStr, "total-"))
	if err != nil {
		fmt.Printf("failed to parse role label %q: %s", numRolesStr, err.Error())

		return nil
	}

	for i := 0; i < numRoles; i++ {
		roleLabel, ok := labels[fmt.Sprintf("%s-%d", types.EMBEDDED_CLUSTER_ROLE_LABEL, i)]
		if !ok {
			fmt.Printf("failed to find role label %d", i)
		}
		toReturn = append(toReturn, roleLabel)
	}

	return toReturn
}
