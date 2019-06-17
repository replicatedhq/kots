package kubernetes

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // register authorization plugins
	"k8s.io/client-go/tools/clientcmd"
)

func NewClient() (kubernetes.Interface, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get client config")
	}
	clientset, err := kubernetes.NewForConfig(config)
	return clientset, errors.Wrap(err, "get kubernetes client")
}
