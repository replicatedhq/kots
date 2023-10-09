package helmvm

import (
	"k8s.io/client-go/kubernetes"
)

func IsHelmVM(clientset kubernetes.Interface) (bool, error) {
	return false, nil
}

func IsHA(clientset kubernetes.Interface) (bool, error) {
	return false, nil
}
