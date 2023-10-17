package helmvm

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/helmvm/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	statsv1alpha1 "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
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

		str, _ := json.Marshal(nodeMetrics)
		fmt.Printf("node %s metrics: %s\n", node.Name, str)

		if nodeMetrics.Usage.Memory() != nil {
			memoryCapacity.Available = float64(nodeMetrics.Usage.Memory().Value()) / math.Pow(2, 30)
		}

		if nodeMetrics.Usage.Cpu() != nil {
			cpuCapacity.Available = cpuCapacity.Capacity - float64(nodeMetrics.Usage.Cpu().Value())
		}

		podCapacity.Available = podCapacity.Capacity - float64(nodeMetrics.Usage.Pods().Value())

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

	body, err := io.ReadAll(r.Body)
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
