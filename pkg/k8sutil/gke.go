package k8sutil

import (
	"strings"

	"k8s.io/client-go/kubernetes"
)

// IsGKEAutopilot returns true if the cluster is positively identified as being an autopilot cluster in GKE
func IsGKEAutopilot(clientset kubernetes.Interface) bool {
	// ignore errors, since resources might be returned anyways
	// ignore groups, since we only need the data contained in resources
	_, resources, _ := clientset.Discovery().ServerGroupsAndResources()
	for _, resource := range resources {
		if strings.HasPrefix(resource.GroupVersion, "auto.gke.io/") {
			return true
		}
	}

	return false
}
