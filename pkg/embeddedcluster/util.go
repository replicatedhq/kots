package embeddedcluster

import (
	"context"
	"fmt"
	"sort"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster-operator/api/v1beta1"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
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

// ClusterConfig will get the list of installations, find the latest installation, and get that installation's config
func ClusterConfig(ctx context.Context) (*embeddedclusterv1beta1.ConfigSpec, error) {
	clientConfig, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster config: %w", err)
	}

	scheme := runtime.NewScheme()
	embeddedclusterv1beta1.AddToScheme(scheme)

	kbClient, err := kbclient.New(clientConfig, kbclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get kubebuilder client: %w", err)
	}

	var installationList embeddedclusterv1beta1.InstallationList
	err = kbClient.List(ctx, &installationList, &kbclient.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list installations: %w", err)
	}

	// determine which of these installations is the latest
	sort.Slice(installationList.Items, func(i, j int) bool {
		return installationList.Items[i].ObjectMeta.CreationTimestamp.After(installationList.Items[j].ObjectMeta.CreationTimestamp.Time)
	})

	latest := installationList.Items[0]
	return latest.Spec.Config, nil
}

// ControllerRoleName determines the name for the 'controller' role
// this might be part of the config, or it might be the default
func ControllerRoleName(ctx context.Context) (string, error) {
	conf, err := ClusterConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster config: %w", err)
	}

	if conf != nil && conf.Controller.Name != "" {
		return conf.Controller.Name, nil
	}
	return DEFAULT_CONTROLLER_ROLE_NAME, nil
}
