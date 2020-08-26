package k8s

import (
	"k8s.io/client-go/kubernetes"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Clientset will return a kubernetes client
func Clientset() (kubernetes.Interface, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func Metricsset() (*metrics.Clientset, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	return metrics.NewForConfig(cfg)
}
