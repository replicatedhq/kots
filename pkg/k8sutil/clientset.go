package k8sutil

import (
	"context"
	"io"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster-kinds/apis/v1beta1"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	DEFAULT_K8S_CLIENT_QPS   = 100
	DEFAULT_K8S_CLIENT_BURST = 100
)

var kubernetesConfigFlags *genericclioptions.ConfigFlags

func init() {
	kubernetesConfigFlags = genericclioptions.NewConfigFlags(false)
	embeddedclusterv1beta1.AddToScheme(scheme.Scheme)
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

func GetK8sVersion(clientset kubernetes.Interface) (string, error) {
	k8sVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", errors.Wrap(err, "failed to get kubernetes server version")
	}
	return k8sVersion.GitVersion, nil
}

func GetK8sMinorVersion(clientset kubernetes.Interface) (int, error) {
	k8sVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return -1, errors.Wrap(err, "failed to get kubernetes server version")
	}

	// remove + sign from Minor version if any (for EKS)
	// https://github.com/aws/containers-roadmap/issues/1404
	reg := regexp.MustCompile(`[0-9]+`)
	minorVersion := reg.FindString(k8sVersion.Minor)
	if minorVersion == "" {
		return -1, errors.New("failed to get k8s minor version")
	}

	k8sMinorVersion, err := strconv.Atoi(minorVersion)
	if err != nil {
		return -1, errors.Wrap(err, "failed to convert k8s minor version to int")
	}
	return k8sMinorVersion, nil
}

func GetDynamicResourceInterface(gvk *schema.GroupVersionKind, namespace string) (dynamic.ResourceInterface, error) {
	config, err := GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	disc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create discovery client")
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disc))

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rest mapping")
	}

	dynamicClientset, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dynamic clientset")
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = dynamicClientset.Resource(mapping.Resource).Namespace(namespace)
	} else {
		dr = dynamicClientset.Resource(mapping.Resource)
	}

	return dr, nil
}

func GetKubeClient(ctx context.Context) (kbclient.Client, error) {
	k8slogger := zap.New(func(o *zap.Options) {
		o.DestWriter = io.Discard
	})
	log.SetLogger(k8slogger)
	cfg, err := GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}
	kcli, err := kbclient.New(cfg, kbclient.Options{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubebuilder client")
	}
	return kcli, nil
}
