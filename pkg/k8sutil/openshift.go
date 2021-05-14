package k8sutil

import (
	"context"
	"strings"

	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// IsOpenShift returns true if the cluster is positively identified as being an openshift cluster
func IsOpenShift(clientset kubernetes.Interface) bool {
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

func OpenShiftVersion() (string, error) {
	// openshift-apiserver does not report version,
	// clusteroperator/openshift-apiserver does, and only version number

	cfg, err := GetClusterConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to get cluster config")
	}

	client, err := configv1client.NewForConfig(cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to build client from config")
	}

	clusterOperator, err := client.ClusterOperators().Get(context.TODO(), "openshift-apiserver", metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get openshift apiserver")
	}

	if clusterOperator == nil {
		return "", errors.New("no openshift apiserver found")
	}

	for _, ver := range clusterOperator.Status.Versions {
		if ver.Name == "operator" {
			return ver.Version, nil
		}
	}

	return "", errors.New("no openshift operator found")
}
