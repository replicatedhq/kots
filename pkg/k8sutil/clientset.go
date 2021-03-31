package k8sutil

import (
	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	KubernetesConfigFlags *genericclioptions.ConfigFlags
)

func GetClientset() (*kubernetes.Clientset, error) {
	cfg, err := GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes clientset")
	}

	return clientset, nil
}

func GetClusterConfig() (*rest.Config, error) {
	var cfg *rest.Config
	var err error

	if KubernetesConfigFlags != nil {
		cfg, err = KubernetesConfigFlags.ToRESTConfig()
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert kube flags to rest config")
		}
	} else {
		cfg, err = config.GetConfig()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get config")
		}
	}

	return cfg, nil
}

func GetK8sVersion() (string, error) {
	clientset, err := GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubernetes clientset")
	}
	k8sVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", errors.Wrap(err, "failed to get kubernetes server version")
	}
	return k8sVersion.GitVersion, nil
}
