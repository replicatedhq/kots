package k8sutil

import (
	"strings"

	"k8s.io/client-go/kubernetes"
)

// IsOpenShift returns true if the cluster is positively identified as being an openshift cluster
func IsOpenShift(clientset *kubernetes.Clientset) bool {
	// ignore errors, since resources might be returned anyways
	// ignore groups, since we only need the data contained in resources
	_, resources, _ := clientset.Discovery().ServerGroupsAndResources()
	if resources != nil {
		for _, resource := range resources {
			if strings.Contains(resource.GroupVersion, "openshift") {
				return true
			}
		}
	}
	return false
}
