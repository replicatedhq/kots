package embeddedcluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"time"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster-kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const configMapName = "embedded-cluster-config"
const configMapNamespace = "embedded-cluster"

// ErrNoInstallations is returned when no installation object is found in the cluster.
var ErrNoInstallations = fmt.Errorf("no installations found")

var (
	chartsArtifactRegex   = regexp.MustCompile(`\/embedded-cluster\/(charts\.tar\.gz):`)
	imagesArtifactRegex   = regexp.MustCompile(`\/embedded-cluster\/(images-.+\.tar):`)
	binaryArtifactRegex   = regexp.MustCompile(`\/embedded-cluster\/(embedded-cluster-.+):`)
	metadataArtifactRegex = regexp.MustCompile(`\/embedded-cluster\/(version-metadata\.json):`)
)

// ReadConfigMap will read the Kurl config from a configmap
func ReadConfigMap(client kubernetes.Interface) (*corev1.ConfigMap, error) {
	return client.CoreV1().ConfigMaps(configMapNamespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
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
	installations, err := ListInstallations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list installations: %w", err)
	}
	if len(installations) == 0 {
		return nil, ErrNoInstallations
	}
	sort.SliceStable(installations, func(i, j int) bool {
		return installations[j].CreationTimestamp.Before(&installations[i].CreationTimestamp)
	})
	return &installations[0], nil
}

func ListInstallations(ctx context.Context) ([]embeddedclusterv1beta1.Installation, error) {
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
	return installationList.Items, nil
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

func getArtifactsFromInstallation(installation kotsv1beta1.Installation, appSlug string) *embeddedclusterv1beta1.ArtifactsLocation {
	if len(installation.Spec.EmbeddedClusterArtifacts) == 0 {
		return nil
	}

	artifacts := &embeddedclusterv1beta1.ArtifactsLocation{}
	for _, artifact := range installation.Spec.EmbeddedClusterArtifacts {
		switch {
		case chartsArtifactRegex.MatchString(artifact):
			artifacts.HelmCharts = artifact
		case imagesArtifactRegex.MatchString(artifact):
			artifacts.Images = artifact
		case binaryArtifactRegex.MatchString(artifact):
			artifacts.EmbeddedClusterBinary = artifact
		case metadataArtifactRegex.MatchString(artifact):
			artifacts.EmbeddedClusterMetadata = artifact
		default:
			logger.Warnf("unknown artifact in installation: %s", artifact)
		}
	}

	return artifacts
}

// startClusterUpgrade will create a new installation with the provided config.
func startClusterUpgrade(ctx context.Context, newcfg embeddedclusterv1beta1.ConfigSpec, artifacts *embeddedclusterv1beta1.ArtifactsLocation) error {
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
			Artifacts:                 artifacts,
			Config:                    &newcfg,
			EndUserK0sConfigOverrides: current.Spec.EndUserK0sConfigOverrides,
		},
	}
	if err := kbClient.Create(ctx, &newins); err != nil {
		return fmt.Errorf("failed to create installation: %w", err)
	}
	return nil
}
