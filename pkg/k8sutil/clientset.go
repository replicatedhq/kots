package k8sutil

import (
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	DEFAULT_K8S_CLIENT_QPS   = 100
	DEFAULT_K8S_CLIENT_BURST = 100
)

var kubernetesConfigFlags *genericclioptions.ConfigFlags

func init() {
	kubernetesConfigFlags = genericclioptions.NewConfigFlags(false)
}

func AddFlags(flags *flag.FlagSet) {
	kubernetesConfigFlags.AddFlags(flags)
}

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

	if kubernetesConfigFlags != nil {
		cfg, err = kubernetesConfigFlags.ToRESTConfig()
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert kube flags to rest config")
		}
	} else {
		cfg, err = config.GetConfig()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get config")
		}
	}

	cfg.QPS = DEFAULT_K8S_CLIENT_QPS
	cfg.Burst = DEFAULT_K8S_CLIENT_BURST

	return cfg, nil
}

func GetDynamicClient() (dynamic.Interface, error) {
	cfg, err := GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dynamic client")
	}
	return dynamicClient, nil
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
