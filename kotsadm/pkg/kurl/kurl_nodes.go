package kurl

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/kurl/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	v1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
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

		podCapacity.Capacity, err = strconv.ParseFloat(node.Status.Capacity.Pods().String(), 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parse pod capacity %q for node %s", node.Status.Capacity.Pods().String(), node.Name)
		}

		// find IP
		nodeIP := ""
		for _, address := range node.Status.Addresses {
			if address.Type == v1.NodeInternalIP {
				nodeIP = address.Address
			}
		}

		if nodeIP == "" {
			logger.Infof("did not find address for node %s, %+v", node.Name, node.Status.Addresses)
		} else {
			nodeMetrics, err := getNodeMetrics(nodeIP)
			if err != nil {
				logger.Infof("got error %s retrieving stats for node %s", err.Error(), node.Name)
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

		toReturn.Nodes = append(toReturn.Nodes, types.Node{
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
func getNodeMetrics(nodeIP string) (*v1alpha1.Summary, error) {
	r := &http.Response{}

	// only use mutual TLS if client cert exists
	_, err := ioutil.ReadFile("/etc/kubernetes/pki/kubelet/client.crt")
	if err == nil {
		cert, err := tls.LoadX509KeyPair("/etc/kubernetes/pki/kubelet/client.crt", "/etc/kubernetes/pki/kubelet/client.key")
		if err != nil {
			return nil, errors.Wrap(err, "get client keypair")
		}

		client := http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates:       []tls.Certificate{cert},
					InsecureSkipVerify: true,
				},
			},
		}
		r, err = client.Get(fmt.Sprintf("https://%s:10250/stats/summary", nodeIP))
	} else {
		client := http.Client{}
		r, err = client.Get(fmt.Sprintf("http://%s:10255/stats/summary", nodeIP))
	}

	if err != nil {
		return nil, errors.Wrapf(err, "get node %s stats", nodeIP)
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "read node %s stats response", nodeIP)
	}

	summary := v1alpha1.Summary{}
	err = json.Unmarshal(body, &summary)
	if err != nil {
		return nil, errors.Wrapf(err, "parse node %s stats response", nodeIP)
	}

	return &summary, nil
}
