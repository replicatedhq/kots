package embeddedcluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

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

// ErrNoInstallations is returned when no installation object is found in the cluster.
var ErrNoInstallations = fmt.Errorf("no installations found")

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

// RequiresUpgrade returns true if the provided configuration differs from the latest active configuration.
func RequiresUpgrade(ctx context.Context, newcfg embeddedclusterv1beta1.ConfigSpec) (bool, error) {
	curcfg, err := ClusterConfig(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current cluster config: %w", err)
	}
	serializedCur, err := json.Marshal(curcfg)
	if err != nil {
		return false, err
	}
	serializedNew, err := json.Marshal(newcfg)
	if err != nil {
		return false, err
	}
	return !bytes.Equal(serializedCur, serializedNew), nil
}

// GetCurrentInstallation returns the most recent installation object from the cluster.
func GetCurrentInstallation(ctx context.Context) (*embeddedclusterv1beta1.Installation, error) {
	clientConfig, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster config: %w", err)
	}
	scheme := runtime.NewScheme()
	embeddedclusterv1beta1.AddToScheme(scheme)
	kbClient, err := kbclient.New(clientConfig, kbclient.Options{Scheme: scheme, WarningHandler: kbclient.WarningHandlerOptions{SuppressWarnings: true}})
	if err != nil {
		return nil, fmt.Errorf("failed to get kubebuilder client: %w", err)
	}
	var installationList embeddedclusterv1beta1.InstallationList
	err = kbClient.List(ctx, &installationList, &kbclient.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list installations: %w", err)
	}
	if len(installationList.Items) == 0 {
		return nil, ErrNoInstallations
	}
	items := installationList.Items
	sort.SliceStable(items, func(i, j int) bool {
		return items[j].CreationTimestamp.Before(&items[i].CreationTimestamp)
	})
	return &installationList.Items[0], nil
}

// ClusterConfig will extract the current cluster configuration from the latest installation
// object found in the cluster.
func ClusterConfig(ctx context.Context) (*embeddedclusterv1beta1.ConfigSpec, error) {
	latest, err := GetCurrentInstallation(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current installation: %w", err)
	}
	return latest.Spec.Config, nil
}

// startClusterUpgrade will create a new installation with the provided config.
func startClusterUpgrade(ctx context.Context, newcfg embeddedclusterv1beta1.ConfigSpec) error {
	clientConfig, err := k8sutil.GetClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get cluster config: %w", err)
	}
	scheme := runtime.NewScheme()
	embeddedclusterv1beta1.AddToScheme(scheme)
	kbClient, err := kbclient.New(clientConfig, kbclient.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("failed to get kubebuilder client: %w", err)
	}
	current, err := GetCurrentInstallation(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current installation: %w", err)
	}
	newins := embeddedclusterv1beta1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name: time.Now().Format("20060102150405"),
		},
		Spec: embeddedclusterv1beta1.InstallationSpec{
			ClusterID:                 current.Spec.ClusterID,
			MetricsBaseURL:            current.Spec.MetricsBaseURL,
			AirGap:                    current.Spec.AirGap,
			Config:                    &newcfg,
			EndUserK0sConfigOverrides: current.Spec.EndUserK0sConfigOverrides,
		},
	}
	if err := kbClient.Create(ctx, &newins); err != nil {
		return fmt.Errorf("failed to create installation: %w", err)
	}
	return nil
}
