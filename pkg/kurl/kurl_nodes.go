package kurl

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kurl/types"
	"github.com/replicatedhq/kots/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	statsv1alpha1 "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

// GetNodes will get a list of nodes with stats
func GetNodes(client kubernetes.Interface) (*types.KurlNodes, error) {
	nodes, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list nodes")
	}

	toReturn := types.KurlNodes{}

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

		nodeIP := ""
		for _, address := range node.Status.Addresses {
			if address.Type == v1.NodeInternalIP {
				nodeIP = address.Address
			}
		}

		if nodeIP == "" {
			logger.Infof("Did not find address for node %s, %+v", node.Name, node.Status.Addresses)
		} else {
			nodeMetrics, err := getNodeMetrics(nodeIP)
			if err != nil {
				logger.Infof("Got error retrieving stats for node %q: %v", node.Name, err)
			} else {
				if nodeMetrics.Node.Memory != nil && nodeMetrics.Node.Memory.AvailableBytes != nil {
					memoryCapacity.Available = float64(*nodeMetrics.Node.Memory.AvailableBytes) / math.Pow(2, 30)
				}

				if nodeMetrics.Node.CPU != nil && nodeMetrics.Node.CPU.UsageNanoCores != nil {
					cpuCapacity.Available = cpuCapacity.Capacity - (float64(*nodeMetrics.Node.CPU.UsageNanoCores) / math.Pow(10, 9))
				}

				podCapacity.Available = podCapacity.Capacity - float64(len(nodeMetrics.Pods))
			}
		}

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

	if val, ok := kurlConf.Data["ha"]; ok && val != "" {
		parsedBool, err := strconv.ParseBool(val)
		if err != nil {
			return nil, errors.Wrapf(err, "parse 'ha' entry in kurl config %q", val)
		}
		toReturn.HA = parsedBool
	}

	return &toReturn, nil
}

func findNodeConditions(conditions []v1.NodeCondition) types.NodeConditions {
	discoveredConditions := types.NodeConditions{}
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

// get kubelet PKI info from /etc/kubernetes/pki/kubelet, use it to hit metrics server at `http://${nodeIP}:10255/stats/summary`
func getNodeMetrics(nodeIP string) (*statsv1alpha1.Summary, error) {
	client := http.Client{
		Timeout: time.Second,
	}
	port := 10255

	// only use mutual TLS if client cert exists
	_, err := os.ReadFile("/etc/kubernetes/pki/kubelet/client.crt")
	if err == nil {
		cert, err := tls.LoadX509KeyPair("/etc/kubernetes/pki/kubelet/client.crt", "/etc/kubernetes/pki/kubelet/client.key")
		if err != nil {
			return nil, errors.Wrap(err, "get client keypair")
		}

		// this will leak memory
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: true,
			},
		}
		port = 10250
	}

	r, err := client.Get(fmt.Sprintf("https://%s:%d/stats/summary", nodeIP, port))
	if err != nil {
		return nil, errors.Wrapf(err, "get node %s stats", nodeIP)
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "read node %s stats response", nodeIP)
	}

	summary := statsv1alpha1.Summary{}
	err = json.Unmarshal(body, &summary)
	if err != nil {
		return nil, errors.Wrapf(err, "parse node %s stats response", nodeIP)
	}

	return &summary, nil
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
			return condition.Status == v1.ConditionTrue
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
		if address.Type == v1.NodeInternalIP {
			return address.Address
		}
	}
	return ""
}
