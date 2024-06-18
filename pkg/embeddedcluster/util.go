package embeddedcluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster-kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	seaweedfsNamespace = "seaweedfs"
	seaweedfsS3SVCName = "ec-seaweedfs-s3"
)

// ErrNoInstallations is returned when no installation object is found in the cluster.
var ErrNoInstallations = fmt.Errorf("no installations found")

func IsHA(clientset kubernetes.Interface) (bool, error) {
	return true, nil
}

// RequiresUpgrade returns true if the provided configuration differs from the latest active configuration.
func RequiresUpgrade(ctx context.Context, kbClient kbclient.Client, newcfg embeddedclusterv1beta1.ConfigSpec) (bool, error) {
	curcfg, err := ClusterConfig(ctx, kbClient)
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
func GetCurrentInstallation(ctx context.Context, kbClient kbclient.Client) (*embeddedclusterv1beta1.Installation, error) {
	installations, err := ListInstallations(ctx, kbClient)
	if err != nil {
		return nil, fmt.Errorf("failed to list installations: %w", err)
	}
	if len(installations) == 0 {
		return nil, ErrNoInstallations
	}
	sort.SliceStable(installations, func(i, j int) bool {
		return installations[j].Name < installations[i].Name
	})
	return &installations[0], nil
}

func ListInstallations(ctx context.Context, kbClient kbclient.Client) ([]embeddedclusterv1beta1.Installation, error) {
	var installationList embeddedclusterv1beta1.InstallationList
	if err := kbClient.List(ctx, &installationList, &kbclient.ListOptions{}); err != nil {
		return nil, fmt.Errorf("failed to list installations: %w", err)
	}
	return installationList.Items, nil
}

// ClusterConfig will extract the current cluster configuration from the latest installation
// object found in the cluster.
func ClusterConfig(ctx context.Context, kbClient kbclient.Client) (*embeddedclusterv1beta1.ConfigSpec, error) {
	latest, err := GetCurrentInstallation(ctx, kbClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get current installation: %w", err)
	}
	return latest.Spec.Config, nil
}

func GetSeaweedFSS3ServiceIP(ctx context.Context, kbClient kbclient.Client) (string, error) {
	nsn := k8stypes.NamespacedName{Name: seaweedfsS3SVCName, Namespace: seaweedfsNamespace}
	var svc corev1.Service
	if err := kbClient.Get(ctx, nsn, &svc); err != nil && !errors.IsNotFound(err) {
		return "", fmt.Errorf("failed to get seaweedfs s3 service: %w", err)
	} else if errors.IsNotFound(err) {
		return "", nil
	}
	return svc.Spec.ClusterIP, nil
}

func getArtifactsFromInstallation(installation kotsv1beta1.Installation, appSlug string) *embeddedclusterv1beta1.ArtifactsLocation {
	if installation.Spec.EmbeddedClusterArtifacts == nil {
		return nil
	}

	return &embeddedclusterv1beta1.ArtifactsLocation{
		EmbeddedClusterBinary:   installation.Spec.EmbeddedClusterArtifacts.BinaryAmd64,
		Images:                  installation.Spec.EmbeddedClusterArtifacts.ImagesAmd64,
		HelmCharts:              installation.Spec.EmbeddedClusterArtifacts.Charts,
		EmbeddedClusterMetadata: installation.Spec.EmbeddedClusterArtifacts.Metadata,
	}
}

// startClusterUpgrade will create a new installation with the provided config.
func startClusterUpgrade(ctx context.Context, newcfg embeddedclusterv1beta1.ConfigSpec, artifacts *embeddedclusterv1beta1.ArtifactsLocation, license kotsv1beta1.License) error {
	kbClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get kubeclient: %w", err)
	}
	current, err := GetCurrentInstallation(ctx, kbClient)
	if err != nil {
		return fmt.Errorf("failed to get current installation: %w", err)
	}
	newins := embeddedclusterv1beta1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name: time.Now().Format("20060102150405"),
			Labels: map[string]string{
				"replicated.com/disaster-recovery": "ec-install",
			},
		},
		Spec: embeddedclusterv1beta1.InstallationSpec{
			ClusterID:                 current.Spec.ClusterID,
			MetricsBaseURL:            current.Spec.MetricsBaseURL,
			HighAvailability:          current.Spec.HighAvailability,
			AirGap:                    current.Spec.AirGap,
			Artifacts:                 artifacts,
			Config:                    &newcfg,
			EndUserK0sConfigOverrides: current.Spec.EndUserK0sConfigOverrides,
			BinaryName:                current.Spec.BinaryName,
			LicenseInfo:               &embeddedclusterv1beta1.LicenseInfo{IsDisasterRecoverySupported: license.Spec.IsDisasterRecoverySupported},
		},
	}
	if err := kbClient.Create(ctx, &newins); err != nil {
		return fmt.Errorf("failed to create installation: %w", err)
	}
	return nil
}
