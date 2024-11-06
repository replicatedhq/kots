package k8sutil

import "k8s.io/client-go/kubernetes"

var _k8s K8sutil

func init() {
	_k8s = &k8sutil{}
}

func Mock(k8s K8sutil) {
	_k8s = k8s
}

type K8sutil interface {
	GetClientset() (kubernetes.Interface, error)
}

// Convenience functions

func GetClientset() (kubernetes.Interface, error) {
	return _k8s.GetClientset()
}
