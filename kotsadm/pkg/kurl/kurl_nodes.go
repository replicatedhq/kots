package kurl

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
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
func GetNodes(client kubernetes.Interface, metrics *metrics.Clientset) (KurlNodes, error) {
	nodes, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return KurlNodes{}, errors.Wrap(err, "list nodes")
	}

	toReturn := KurlNodes{}

	for _, node := range nodes.Items {
		cpuCapacity, _ := strconv.ParseFloat(node.Status.Capacity.Cpu().String(), 64)
		memCapacity := float64(node.Status.Capacity.Memory().Value()) / 1000000000 // capacity in GB
		podCapacity, _ := strconv.ParseFloat(node.Status.Capacity.Pods().String(), 64)

		toReturn.Nodes = append(toReturn.Nodes, Node{
			Name:           node.Name,
			IsConnected:    true,
			CanDelete:      node.Spec.Unschedulable,
			KubeletVersion: node.Status.NodeInfo.KubeletVersion,
			CPU: CapacityAvailable{
				Capacity: cpuCapacity, // TODO include available resources
			},
			Memory: CapacityAvailable{
				Capacity: memCapacity,
			},
			Pods: CapacityAvailable{
				Capacity: podCapacity,
			},
			Conditions: findNodeConditions(node.Status.Conditions),
		})
	}

	kurlConf, err := ReadConfigMap(client)
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			toReturn.IsKurlEnabled = false
			return toReturn, nil
		}
		return KurlNodes{}, errors.Wrap(err, "get kurl config")
	} else {
		toReturn.IsKurlEnabled = true
	}

	if val, ok := kurlConf.Data["ha"]; ok {
		parsedBool, err := strconv.ParseBool(val)
		if err != nil {
			return KurlNodes{}, errors.Wrapf(err, "parse 'ha' entry in kurl config %q", val)
		}
		toReturn.HA = parsedBool
	}

	return toReturn, nil
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
