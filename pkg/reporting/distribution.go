package reporting

import (
	"context"
	"strings"

	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetDistribution(clientset kubernetes.Interface) Distribution {
	// First try get the special ones. This is because sometimes we cannot get the distribution from the server version
	if distribution := distributionFromServerGroupAndResources(clientset); distribution != UnknownDistribution {
		return distribution
	}

	if distribution := distributionFromProviderId(clientset); distribution != UnknownDistribution {
		return distribution
	}

	if distribution := distributionFromLabels(clientset); distribution != UnknownDistribution {
		return distribution
	}

	// Getting distribution from server version string
	k8sVersion, err := k8sutil.GetK8sVersion(clientset)
	if err != nil {
		logger.Debugf("failed to get k8s version: %v", err.Error())
		return UnknownDistribution
	}
	if distribution := distributionFromVersion(k8sVersion); distribution != UnknownDistribution {
		return distribution
	}

	return UnknownDistribution
}

func distributionFromServerGroupAndResources(clientset kubernetes.Interface) Distribution {
	_, resources, _ := clientset.Discovery().ServerGroupsAndResources()
	for _, resource := range resources {
		switch {
		case strings.HasPrefix(resource.GroupVersion, "apps.openshift.io/"):
			return OpenShift
		case strings.HasPrefix(resource.GroupVersion, "run.tanzu.vmware.com/"):
			return Tanzu
		}
	}

	return UnknownDistribution
}

func distributionFromProviderId(clientset kubernetes.Interface) Distribution {
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	nodeCount := len(nodes.Items)
	if nodeCount > 1 {
		logger.Infof("Found %d nodes", nodeCount)
	} else {
		logger.Infof("Found %d node", nodeCount)
	}
	if err != nil {
		logger.Infof("got error listing node: %v", err.Error())
	}
	if len(nodes.Items) >= 1 {
		node := nodes.Items[0]
		if strings.HasPrefix(node.Spec.ProviderID, "kind:") {
			return Kind
		}
		if strings.HasPrefix(node.Spec.ProviderID, "digitalocean:") {
			return DigitalOcean
		}
	}
	return UnknownDistribution
}

func distributionFromLabels(clientset kubernetes.Interface) Distribution {
	nodes, _ := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	for _, node := range nodes.Items {
		for k, v := range node.ObjectMeta.Labels {
			if k == "kurl.sh/cluster" && v == "true" {
				return Kurl
			} else if k == "microk8s.io/cluster" && v == "true" {
				return MicroK8s
			}
			if k == "kubernetes.azure.com/role" {
				return AKS
			}
			if k == "minikube.k8s.io/version" {
				return Minikube
			}
			if k == "oci.oraclecloud.com/fault-domain" {
				// Based on: https://docs.oracle.com/en-us/iaas/Content/ContEng/Reference/contengsupportedlabelsusecases.htm
				return OKE
			}
			if k == "kots.io/embedded-cluster-role" {
				return EmbeddedCluster
			}
		}
	}
	return UnknownDistribution
}

func distributionFromVersion(k8sVersion string) Distribution {
	switch {
	case strings.Contains(k8sVersion, "-gke."):
		return GKE
	case strings.Contains(k8sVersion, "-eks-"):
		return EKS
	case strings.Contains(k8sVersion, "+rke2"):
		return RKE2
	case strings.Contains(k8sVersion, "+k3s"):
		return K3s
	case strings.Contains(k8sVersion, "+k0s"):
		return K0s
	default:
		return UnknownDistribution
	}
}
