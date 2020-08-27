package kurl

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/k8s"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	v1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KurlNodes struct {
	Nodes         []Node `json:"nodes"`
	HA            bool   `json:"ha"`
	IsKurlEnabled bool   `json:"isKurlEnabled"`
}

type Node struct {
	Name           string            `json:"name"`
	IsConnected    bool              `json:"isConnected"`
	CanDelete      bool              `json:"canDelete"`
	KubeletVersion string            `json:"kubeletVersion"`
	CPU            CapacityAvailable `json:"cpu"`
	Memory         CapacityAvailable `json:"memory"`
	Pods           CapacityAvailable `json:"pods"`
	Conditions     NodeConditions    `json:"conditions"`
}

type CapacityAvailable struct {
	Capacity  float64 `json:"capacity"`
	Available float64 `json:"available"`
}

type NodeConditions struct {
	MemoryPressure bool `json:"memoryPressure"`
	DiskPressure   bool `json:"diskPressure"`
	PidPressure    bool `json:"pidPressure"`
	Ready          bool `json:"ready"`
}

// GetNodes will get a list of nodes with stats
func GetNodes(client kubernetes.Interface) (*KurlNodes, error) {
	nodes, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list nodes")
	}

	mc, err := k8s.Metricsset()
	if err != nil {
		return nil, errors.Wrap(err, "get metrics client")
	}

	toReturn := KurlNodes{}

	for _, node := range nodes.Items {
		cpuCapacity := CapacityAvailable{}
		memoryCapacity := CapacityAvailable{}
		podCapacity := CapacityAvailable{}

		memoryCapacity.Capacity = float64(node.Status.Capacity.Memory().Value()) / 1000000000 // capacity in GB

		cpuCapacity.Capacity, err = strconv.ParseFloat(node.Status.Capacity.Cpu().String(), 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parse CPU capacity %q for node %s", node.Status.Capacity.Cpu().String(), node.Name)
		}

		podCapacity.Capacity, err = strconv.ParseFloat(node.Status.Capacity.Pods().String(), 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parse pod capacity %q for node %s", node.Status.Capacity.Pods().String(), node.Name)
		}

		nodeMetrics, err := mc.MetricsV1beta1().NodeMetricses().Get(context.TODO(), node.Name, metav1.GetOptions{})
		if err != nil {
			logger.Infof("got error %s retrieving stats for node %s", err.Error(), node.Name)
		} else {
			memoryCapacity.Available = memoryCapacity.Capacity - (float64(nodeMetrics.Usage.Memory().Value()) / 1000000000)

			cpuCapacity.Available, err = strconv.ParseFloat(nodeMetrics.Usage.Cpu().String(), 64)
			if err != nil {
				return nil, errors.Wrapf(err, "parse CPU available %q for node %s", nodeMetrics.Usage.Cpu().String(), node.Name)
			}
			cpuCapacity.Available = cpuCapacity.Capacity - cpuCapacity.Available

			nodeMetrics.Usage.Pods()

			podCapacity.Available, err = strconv.ParseFloat(nodeMetrics.Usage.Pods().String(), 64)
			if err != nil {
				return nil, errors.Wrapf(err, "parse pods available %q for node %s", nodeMetrics.Usage.Pods().String(), node.Name)
			}
			podCapacity.Available = podCapacity.Capacity - podCapacity.Available
		}

		toReturn.Nodes = append(toReturn.Nodes, Node{
			Name:           node.Name,
			IsConnected:    true,
			CanDelete:      node.Spec.Unschedulable,
			KubeletVersion: node.Status.NodeInfo.KubeletVersion,
			CPU:            cpuCapacity,
			Memory:         memoryCapacity,
			Pods:           podCapacity,
			Conditions:     findNodeConditions(node.Status.Conditions),
		})
	}

	kurlConf, err := ReadConfigMap(client)
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			toReturn.IsKurlEnabled = false
			return &toReturn, nil
		}
		return nil, errors.Wrap(err, "get kurl config")
	}

	toReturn.IsKurlEnabled = true

	if val, ok := kurlConf.Data["ha"]; ok {
		parsedBool, err := strconv.ParseBool(val)
		if err != nil {
			return nil, errors.Wrapf(err, "parse 'ha' entry in kurl config %q", val)
		}
		toReturn.HA = parsedBool
	}

	return &toReturn, nil
}

func findNodeConditions(conditions []v1.NodeCondition) NodeConditions {
	discoveredConditions := NodeConditions{}
	for _, condition := range conditions {
		if condition.Type == "MemoryPressure" {
			discoveredConditions.MemoryPressure = condition.Status == v1.ConditionTrue
		}
		if condition.Type == "DiskPressure" {
			discoveredConditions.DiskPressure = condition.Status == v1.ConditionTrue
		}
		if condition.Type == "PIDPressure" {
			discoveredConditions.PidPressure = condition.Status == v1.ConditionTrue
		}
		if condition.Type == "Ready" {
			discoveredConditions.Ready = condition.Status == v1.ConditionTrue
		}
	}
	return discoveredConditions
}
