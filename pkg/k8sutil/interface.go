package k8sutil

import "k8s.io/client-go/kubernetes"

var _k8s K8sutilInterface

func init() {
	Set(&K8sutil{})
}

func Set(k8s K8sutilInterface) {
	_k8s = k8s
}

type K8sutilInterface interface {
	GetClientset() (kubernetes.Interface, error)
}

// Convenience functions

func GetClientset() (kubernetes.Interface, error) {
	return _k8s.GetClientset()
}
