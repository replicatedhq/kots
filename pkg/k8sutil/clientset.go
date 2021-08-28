package k8sutil

import (
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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
	if kubernetesConfigFlags != nil {
		cfg, err := kubernetesConfigFlags.ToRESTConfig()
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert kube flags to rest config")
		}
		return cfg, nil
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config")
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

// Implements Helm v3 RESTClientGetter interface
type RESTClientGetter struct {
}

func (c RESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	config, err := GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	return config, nil
}

func (c RESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	clientset, err := GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes clientset")
	}

	return &CachedDiscoveryClient{clientSet: clientset}, nil
}

func (c RESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	clientset, err := GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes clientset")
	}

	discoveryClient := &CachedDiscoveryClient{clientSet: clientset}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient)
	return expander, nil
}

func (c RESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return kubernetesConfigFlags.ToRawKubeConfigLoader()
}
