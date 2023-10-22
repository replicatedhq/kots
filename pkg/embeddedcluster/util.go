package embeddedcluster

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const configMapName = "embedded-cluster-config"
const configMapNamespace = "embedded-cluster"

// ReadConfigMap will read the Kurl config from a configmap
func ReadConfigMap(client kubernetes.Interface) (*corev1.ConfigMap, error) {
	return client.CoreV1().ConfigMaps(configMapNamespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
}

func IsEmbeddedCluster(clientset kubernetes.Interface) (bool, error) {
	if clientset == nil {
		return false, fmt.Errorf("clientset is nil")
	}

	configMapExists := false
	_, err := ReadConfigMap(clientset)
	if err == nil {
		configMapExists = true
	} else if kuberneteserrors.IsNotFound(err) {
		configMapExists = false
	} else if kuberneteserrors.IsUnauthorized(err) {
		configMapExists = false
	} else if kuberneteserrors.IsForbidden(err) {
		configMapExists = false
	} else if err != nil {
		return false, fmt.Errorf("failed to get embedded cluster configmap: %w", err)
	}

	return configMapExists, nil
}

func IsHA(clientset kubernetes.Interface) (bool, error) {
	return true, nil
}

func ClusterID(client kubernetes.Interface) (string, error) {
	configMap, err := ReadConfigMap(client)
	if err != nil {
		return "", fmt.Errorf("failed to read configmap: %w", err)
	}

	return configMap.Data["embedded-cluster-id"], nil
}
