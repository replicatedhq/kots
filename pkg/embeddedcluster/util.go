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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
	k8syaml "sigs.k8s.io/yaml"
)

const configMapName = "embedded-cluster-config"
const configMapNamespace = "embedded-cluster"

// ErrNoInstallations is returned when no installation object is found in the cluster.
var ErrNoInstallations = fmt.Errorf("no installations found")

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
func RequiresUpgrade(ctx context.Context, rawcfg string) (bool, error) {
	in, err := GetCurrentInstallation(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current installation: %w", err)
	}

	// we can't determine the current cluster configuration so let's just start an upgrade.
	// this may be an noops if the configuration is the same in the end.
	if in.Spec.Config == nil && in.Spec.ConfigSecret == nil {
		return true, nil
	}

	// start by unmarshaling all known configs to validate the config is a valid yaml.
	if err := k8syaml.Unmarshal([]byte(rawcfg), &embeddedclusterv1beta1.Config{}); err != nil {
		return false, fmt.Errorf("failed to unmarshal new cluster config: %w", err)
	}

	// if there is no config in the installation then it is stored in a secret. we read it
	// from there and compare with the new config.
	if in.Spec.Config == nil {
		curcfg, err := readClusterConfigFromSecret(ctx, in)
		if err != nil {
			return false, fmt.Errorf("failed to read current cluster configuration: %w", err)
		}
		differs, err := configSpecDiffers(string(curcfg), rawcfg)
		if err != nil {
			return false, fmt.Errorf("failed comparing config differences: %w", err)
		}
		return differs, nil
	}

	// we do an strict unmarshal to capture the presence of field unknown to the version of
	// the crd we are using. if this fails then we consider an upgrade necessary. we already
	// know it to be a valid yaml.
	var newcfg embeddedclusterv1beta1.Config
	if err := k8syaml.UnmarshalStrict([]byte(rawcfg), &newcfg); err != nil {
		return true, nil
	}

	// if the config is set in the installation object then we compare it with the new config.
	serializedCur, err := json.Marshal(in.Spec.Config)
	if err != nil {
		return false, fmt.Errorf("failed to marshal current cluster config: %w", err)
	}
	serializedNew, err := json.Marshal(newcfg.Spec)
	if err != nil {
		return false, fmt.Errorf("failed to marshal new cluster config: %w", err)
	}
	return !bytes.Equal(serializedCur, serializedNew), nil
}

// configSpecDiffers compares if the spec property of the provided yamls differs one from
// another. returns true if they differ, false otherwise.
func configSpecDiffers(a, b string) (bool, error) {
	type Wrapper struct {
		Spec map[string]interface{} `yaml:"spec"`
	}

	var aMap Wrapper
	if err := k8syaml.Unmarshal([]byte(a), &aMap); err != nil {
		return false, fmt.Errorf("yaml A values error: %w", err)
	}

	var bMap Wrapper
	if err := k8syaml.Unmarshal([]byte(b), &bMap); err != nil {
		return false, fmt.Errorf("yaml B values error: %w", err)
	}

	aYaml, err := k8syaml.Marshal(aMap.Spec)
	if err != nil {
		return false, fmt.Errorf("yaml A marshal error: %w", err)
	}

	bYaml, err := k8syaml.Marshal(bMap.Spec)
	if err != nil {
		return false, fmt.Errorf("yaml B marshal error: %w", err)
	}

	return string(aYaml) != string(bYaml), nil
}

// readClusterConfigFromSecret reads the cluster config for the provided installation. this
// function may return a nil without error if the secret does not contain the cluster config
// in the expected index.
func readClusterConfigFromSecret(ctx context.Context, in *embeddedclusterv1beta1.Installation) ([]byte, error) {
	if in.Spec.ConfigSecret == nil {
		return nil, fmt.Errorf("installation does not have a config secret")
	}

	kbClient, err := k8sutil.GetControllerRuntimeClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get controller runtime client: %w", err)
	}

	nsn := types.NamespacedName{Namespace: in.Spec.ConfigSecret.Namespace, Name: in.Spec.ConfigSecret.Name}
	var secret corev1.Secret
	if err := kbClient.Get(ctx, nsn, &secret); err != nil {
		return nil, fmt.Errorf("failed to get secret config secret: %w", err)
	}

	return secret.Data[embeddedclusterv1beta1.ConfigSecretEntryName], nil
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
		return installations[j].Name < installations[i].Name
	})
	return &installations[0], nil
}

func ListInstallations(ctx context.Context) ([]embeddedclusterv1beta1.Installation, error) {
	kbClient, err := k8sutil.GetControllerRuntimeClient()
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
// object found in the cluster. If the installation object points to a secret the config is
// read from there.
func ClusterConfig(ctx context.Context) (*embeddedclusterv1beta1.ConfigSpec, error) {
	latest, err := GetCurrentInstallation(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current installation: %w", err)
	}

	// if the cluster does not have a config secret we return the config from the
	// installation object. this is for backward compatibility.
	if latest.Spec.ConfigSecret == nil {
		return latest.Spec.Config, nil
	}

	rawConfig, err := readClusterConfigFromSecret(ctx, latest)
	if err != nil {
		return nil, fmt.Errorf("failed to read cluster config from secret: %w", err)
	} else if len(rawConfig) == 0 {
		return nil, fmt.Errorf("cluster config secret is empty")
	}

	var config embeddedclusterv1beta1.Config
	if err := k8syaml.Unmarshal(rawConfig, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cluster config: %w", err)
	}
	return &config.Spec, nil
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
func startClusterUpgrade(ctx context.Context, cfg string, artifacts *embeddedclusterv1beta1.ArtifactsLocation, license kotsv1beta1.License) error {
	kbClient, err := k8sutil.GetControllerRuntimeClient()
	if err != nil {
		return fmt.Errorf("failed to get kubebuilder client: %w", err)
	}
	current, err := GetCurrentInstallation(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current installation: %w", err)
	}

	// we unmarhsal the full configuration, we will only use the spec portion.
	var newcfg embeddedclusterv1beta1.Config
	if err := k8syaml.Unmarshal([]byte(cfg), &newcfg); err != nil {
		return fmt.Errorf("failed to unmarshal new config: %w", err)
	}
	cfgspec := &newcfg.Spec

	// we now attempt to do a strict unmarshal to detect the presence of unknown
	// fields in the crd. if we have those then we need to store the full config
	// in a secret.
	installationName := time.Now().Format("20060102150405")
	var configSecret *embeddedclusterv1beta1.ConfigSecret //
	if err := k8syaml.UnmarshalStrict([]byte(cfg), &embeddedclusterv1beta1.Config{}); err != nil {
		clusterConfigName := fmt.Sprintf("cluster-config-%s", installationName)
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterConfigName,
				Namespace: "embedded-cluster",
			},
			Data: map[string][]byte{
				embeddedclusterv1beta1.ConfigSecretEntryName: []byte(cfg),
			},
		}
		if err := kbClient.Create(ctx, &secret); err != nil {
			return fmt.Errorf("failed to create cluster config secret: %w", err)
		}
		cfgspec = nil
		configSecret = &embeddedclusterv1beta1.ConfigSecret{
			Name:      clusterConfigName,
			Namespace: "embedded-cluster",
		}
	}

	newins := embeddedclusterv1beta1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name: installationName,
			Labels: map[string]string{
				"replicated.com/disaster-recovery": "ec-install",
			},
		},
		Spec: embeddedclusterv1beta1.InstallationSpec{
			ClusterID:                 current.Spec.ClusterID,
			MetricsBaseURL:            current.Spec.MetricsBaseURL,
			AirGap:                    current.Spec.AirGap,
			Artifacts:                 artifacts,
			Config:                    cfgspec,
			EndUserK0sConfigOverrides: current.Spec.EndUserK0sConfigOverrides,
			BinaryName:                current.Spec.BinaryName,
			LicenseInfo:               &embeddedclusterv1beta1.LicenseInfo{IsSnapshotSupported: license.Spec.IsSnapshotSupported},
			ConfigSecret:              configSecret,
		},
	}
	if err := kbClient.Create(ctx, &newins); err != nil {
		return fmt.Errorf("failed to create installation: %w", err)
	}
	return nil
}
