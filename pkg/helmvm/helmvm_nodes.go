package helmvm

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/helmvm/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// GetNodes will get a list of nodes with stats
func GetNodes(ctx context.Context, client kubernetes.Interface) (*types.HelmVMNodes, error) {
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

	toReturn := types.HelmVMNodes{}

	nodePods, err := podsPerNode(ctx, client)
	if err != nil {
		return nil, errors.Wrap(err, "pods per node")
	}

	for _, node := range nodes.Items {
		cpuCapacity := types.CapacityAvailable{}
		memoryCapacity := types.CapacityAvailable{}
		podCapacity := types.CapacityAvailable{}

		memoryCapacity.Capacity = float64(node.Status.Capacity.Memory().Value()) / math.Pow(2, 30) // capacity in GB

		cpuCapacity.Capacity, err = strconv.ParseFloat(node.Status.Capacity.Cpu().String(), 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parse CPU capacity %q for node %s", node.Status.Capacity.Cpu().String(), node.Name)
		}

		podCapacity.Capacity = float64(node.Status.Capacity.Pods().Value())

		nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, node.Name, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "list pod metrics")
		}

		if nodeMetrics.Usage.Memory() != nil {
			memoryCapacity.Available = memoryCapacity.Capacity - float64(nodeMetrics.Usage.Memory().Value())/math.Pow(2, 30)
		}

		if nodeMetrics.Usage.Cpu() != nil {
			cpuCapacity.Available = cpuCapacity.Capacity - nodeMetrics.Usage.Cpu().AsApproximateFloat64()
		}

		podCapacity.Available = podCapacity.Capacity - float64(len(nodePods[node.Name]))

		nodeLabelArray := []string{}
		for k, v := range node.Labels {
			nodeLabelArray = append(nodeLabelArray, fmt.Sprintf("%s:%s", k, v))
		}

		toReturn.Nodes = append(toReturn.Nodes, types.Node{
			Name:           node.Name,
			IsConnected:    isConnected(node),
			IsReady:        isReady(node),
			IsPrimaryNode:  isPrimary(node),
			CanDelete:      node.Spec.Unschedulable && !isConnected(node),
			KubeletVersion: node.Status.NodeInfo.KubeletVersion,
			CPU:            cpuCapacity,
			Memory:         memoryCapacity,
			Pods:           podCapacity,
			Labels:         nodeLabelArray,
			Conditions:     findNodeConditions(node.Status.Conditions),
			PodList:        nodePods[node.Name],
		})
	}

	isHelmVM, err := IsHelmVM(client)
	if err != nil {
		return nil, errors.Wrap(err, "is helmvm")
	}
	toReturn.IsHelmVMEnabled = isHelmVM

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

// podsPerNode returns a map of node names to pods, across all namespaces
func podsPerNode(ctx context.Context, client kubernetes.Interface) (map[string][]corev1.Pod, error) {
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list namespaces")
	}

	toReturn := map[string][]corev1.Pod{}

	for _, ns := range namespaces.Items {
		nsPods, err := client.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "list pods in namespace %s", ns.Name)
		}

		for _, pod := range nsPods.Items {
			pod := pod
			if pod.Spec.NodeName == "" {
				continue
			}

			if _, ok := toReturn[pod.Spec.NodeName]; !ok {
				toReturn[pod.Spec.NodeName] = []corev1.Pod{}
			}

			toReturn[pod.Spec.NodeName] = append(toReturn[pod.Spec.NodeName], pod)
		}
	}

	return toReturn, nil
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

func internalIP(node corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			return address.Address
		}
	}
	return ""
}
