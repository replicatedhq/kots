package embeddedcluster

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SeaweedfsNamespace = "seaweedfs"
	SeaweedfsS3SVCName = "ec-seaweedfs-s3"
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
		// if there is no installation object we can't start an upgrade. this is a valid
		// scenario specially during cluster bootstrap. as we do not need to upgrade the
		// cluster just after its installation we can return nil here.
		// (the cluster in the first kots version will match the cluster installed during bootstrap)
		if errors.Is(err, ErrNoInstallations) {
			return false, nil
		}
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

func InstallationSucceeded(ctx context.Context, ins *embeddedclusterv1beta1.Installation) bool {
	return ins.Status.State == embeddedclusterv1beta1.InstallationStateInstalled
}

func InstallationFailed(ctx context.Context, ins *embeddedclusterv1beta1.Installation) bool {
	switch ins.Status.State {
	case embeddedclusterv1beta1.InstallationStateFailed,
		embeddedclusterv1beta1.InstallationStateHelmChartUpdateFailure,
		embeddedclusterv1beta1.InstallationStateObsolete:
		return true
	}
	return false
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
	nsn := k8stypes.NamespacedName{Name: SeaweedfsS3SVCName, Namespace: SeaweedfsNamespace}
	var svc corev1.Service
	if err := kbClient.Get(ctx, nsn, &svc); err != nil && !k8serrors.IsNotFound(err) {
		return "", fmt.Errorf("failed to get seaweedfs s3 service: %w", err)
	} else if k8serrors.IsNotFound(err) {
		return "", nil
	}
	return svc.Spec.ClusterIP, nil
}

func getArtifactsFromInstallation(installation kotsv1beta1.Installation) *embeddedclusterv1beta1.ArtifactsLocation {
	if installation.Spec.EmbeddedClusterArtifacts == nil {
		return nil
	}

	return &embeddedclusterv1beta1.ArtifactsLocation{
		EmbeddedClusterBinary:   installation.Spec.EmbeddedClusterArtifacts.BinaryAmd64,
		Images:                  installation.Spec.EmbeddedClusterArtifacts.ImagesAmd64,
		HelmCharts:              installation.Spec.EmbeddedClusterArtifacts.Charts,
		EmbeddedClusterMetadata: installation.Spec.EmbeddedClusterArtifacts.Metadata,
		AdditionalArtifacts:     installation.Spec.EmbeddedClusterArtifacts.AdditionalArtifacts,
	}
}
