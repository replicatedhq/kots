package snapshot

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	types "github.com/replicatedhq/kots/pkg/snapshot/types"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	FileSystemLVPConfigMapName = "kotsadm-fs-lvp"
)

func DeployFileSystemLvp(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryConfig kotsadmtypes.RegistryConfig) error {
	veleroNamespace, err := DetectVeleroNamespace(ctx, clientset, deployOptions.Namespace)
	if err != nil {
		return errors.Wrap(err, "failed to detect velero namespace")
	}

	// Ensure the config map exists for the plugin
	if err = EnsureLocalVolumeProviderConfigMaps(deployOptions, veleroNamespace); err != nil {
		return errors.Wrap(err, "failed to configure local volume provider plugin config map")
	}

	return nil
}

func ValidateFileSystemDeployment(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryConfig kotsadmtypes.RegistryConfig) (bool, bool, error) {
	// configure fs directory/mount. This is a legacy check to see if this directory was migrated from Minio and has an intermediate directory
	_, hasMinioConfig, writable, err := shouldResetFileSystemMinioMount(ctx, clientset, deployOptions, registryConfig)
	if err != nil {
		return false, false, errors.Wrap(err, "failed to check if should reset file system mount")
	}
	return hasMinioConfig, writable, nil
}

func GetCurrentLvpFileSystemConfig(ctx context.Context, namespace string) (*types.FileSystemConfig, error) {
	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero clientset")
	}

	bsl, err := FindBackupStoreLocation(ctx, clientset, veleroClient, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find velero backup storage location")
	}
	if bsl != nil {
		// backup storage location exists, get file system config from there
		switch bsl.Spec.Provider {
		case SnapshotStoreHostPathProvider:
			fileSystemConfig := &types.FileSystemConfig{}
			hostPath := bsl.Spec.Config["path"]
			fileSystemConfig.HostPath = &hostPath
			return fileSystemConfig, nil
		case SnapshotStoreNFSProvider:
			fileSystemConfig := &types.FileSystemConfig{
				NFS: &types.NFSConfig{
					Path:   bsl.Spec.Config["path"],
					Server: bsl.Spec.Config["server"],
				},
			}
			return fileSystemConfig, nil
		}
		return nil, nil
	}

	// backup storage location does not exist, get file system config from the config map
	fileSystemConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, FileSystemLVPConfigMapName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get file system configmap")
	}

	if fileSystemConfigMap.Data == nil {
		return nil, nil
	}

	fileSystemConfig := types.FileSystemConfig{}

	if hostPath, ok := fileSystemConfigMap.Data["HOSTPATH"]; ok {
		fileSystemConfig.HostPath = &hostPath
	} else if _, ok := fileSystemConfigMap.Data["NFS_PATH"]; ok {
		fileSystemConfig.NFS = &types.NFSConfig{
			Path:   fileSystemConfigMap.Data["NFS_PATH"],
			Server: fileSystemConfigMap.Data["NFS_SERVER"],
		}
	}

	return &fileSystemConfig, nil
}

// RevertToMinioFS will apply the spec of the previous BSL to the current one and then update.
// Used for recovery during a failed migration from Minio to LVP.
func RevertToMinioFS(ctx context.Context, clientset kubernetes.Interface, veleroClient veleroclientv1.VeleroV1Interface, kotsadmNamespace, veleroNamespace string, previousBsl *velerov1api.BackupStorageLocation) error {
	bsl, err := FindBackupStoreLocation(context.TODO(), clientset, veleroClient, kotsadmNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to find backupstoragelocations")
	}
	if bsl == nil {
		return errors.New("backup storage location not found")
	}

	bsl.Spec = previousBsl.Spec

	err = UpdateBackupStorageLocation(ctx, veleroNamespace, bsl)
	if err != nil {
		return errors.Wrap(err, "failed to revert to minio backup storage location")
	}
	return nil
}

// DeleteFileSystemMinio cleans up the minio resources for hostpath and nfs snapshot deployments.
// The secret is not deleted, just in case.
func DeleteFileSystemMinio(ctx context.Context, kotsadmNamespace string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes clientset")
	}

	if err := clientset.CoreV1().ConfigMaps(kotsadmNamespace).Delete(ctx, FileSystemMinioConfigMapName, metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "failed to delete fs minio config map")
	}

	if err := clientset.AppsV1().Deployments(kotsadmNamespace).Delete(ctx, FileSystemMinioDeploymentName, metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "failed to delete fs minio deployment")
	}

	if err := clientset.CoreV1().Services(kotsadmNamespace).Delete(ctx, FileSystemMinioServiceName, metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "failed to delete fs minio service")
	}

	return nil
}

// EnsureLocalVolumeProviderConfigMaps ensures two configmaps:
// one customizes the LVP plugin deployment with a config map based on the chosen file system backing and the detection of Openshift.
// This ensures that Velero and NodeAgent have permissions to write to the disk.
// the second config map is used to store the current file system configuration.
func EnsureLocalVolumeProviderConfigMaps(deployOptions FileSystemDeployOptions, veleroNamespace string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes clientset")
	}

	if err := ensureFileSystemLVPConfigMap(context.TODO(), clientset, deployOptions); err != nil {
		return errors.Wrap(err, "failed to ensure file system lvp config map")
	}

	if deployOptions.IsOpenShift || veleroNamespace == "" {
		return nil
	}

	fsConfig := deployOptions.FileSystemConfig

	bucket, err := GetLvpBucket(&fsConfig)
	if err != nil {
		return errors.Wrap(err, "failed to get lvp bucket")
	}

	var pluginConfigMapLabel string
	if fsConfig.HostPath != nil {
		pluginConfigMapLabel = SnapshotStoreHostPathProvider
	} else {
		pluginConfigMapLabel = SnapshotStoreNFSProvider
	}

	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", pluginConfigMapLabel, "ObjectStore"),
	}

	configmaps, err := clientset.CoreV1().ConfigMaps(veleroNamespace).List(context.TODO(), listOpts)
	if err != nil {
		return errors.Wrap(err, "failed to list existing config maps")
	}

	if len(configmaps.Items) == 0 {
		// Create the config map
		configmap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "local-volume-provider-config",
				Namespace: veleroNamespace,
				Labels: map[string]string{
					"velero.io/plugin-config":     "",
					SnapshotStoreNFSProvider:      "ObjectStore",
					SnapshotStoreHostPathProvider: "ObjectStore",
				},
			},
			// These values are the settings used for the minio filesystem deployment
			Data: map[string]string{
				"securityContextRunAsUser": "1001",
				"securityContextFsGroup":   "1001",
				"preserveVolumes":          bucket,
			},
		}

		_, err = clientset.CoreV1().ConfigMaps(veleroNamespace).Create(context.TODO(), configmap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create new local-volume-provider config map")
		}

		return nil
	}

	configmap := &configmaps.Items[0]
	configmap.Data["securityContextRunAsUser"] = "1001"
	configmap.Data["securityContextFsGroup"] = "1001"
	configmap.Data["preserveVolumes"] = bucket

	_, err = clientset.CoreV1().ConfigMaps(veleroNamespace).Update(context.TODO(), configmap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update local-volume-provider config map")
	}

	return nil
}

func ensureFileSystemLVPConfigMap(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions) error {
	configmap := fileSystemLVPConfigMapResource(deployOptions.FileSystemConfig)

	existingConfigMap, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Get(ctx, configmap.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing configmap")
		}

		_, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Create(ctx, configmap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create configmap")
		}

		return nil
	}

	existingConfigMap = updateFileSystemLVPConfigMap(existingConfigMap, configmap)

	_, err = clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Update(ctx, existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update deployment")
	}

	return nil
}

func fileSystemLVPConfigMapResource(fileSystemConfig types.FileSystemConfig) *corev1.ConfigMap {
	data := map[string]string{}

	if fileSystemConfig.HostPath != nil {
		data["HOSTPATH"] = *fileSystemConfig.HostPath
	} else if fileSystemConfig.NFS != nil {
		data["NFS_PATH"] = fileSystemConfig.NFS.Path
		data["NFS_SERVER"] = fileSystemConfig.NFS.Server
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: FileSystemLVPConfigMapName,
		},
		Data: data,
	}
}

func updateFileSystemLVPConfigMap(existingConfigMap, desiredConfigMap *corev1.ConfigMap) *corev1.ConfigMap {
	existingConfigMap.Data = desiredConfigMap.Data

	return existingConfigMap
}

// IsFileSystemMinioDisable returns the value of an internal KOTS config map entry indicating
// if this installation has opted in or out of migrating from Minio to the LVP plugin.
func IsFileSystemMinioDisabled(kotsadmNamespace string) (bool, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return false, errors.Wrap(err, "failed to get kubernetes clientset")
	}

	//Minio disabled is detected based on two cases
	// 1. minio image is not present in the cluster
	// 2. disableS3 flag is enabled
	minioImage, err := image.GetMinioImage(clientset, kotsadmNamespace)
	if err != nil {
		return false, errors.Wrap(err, "failed to check minio image")
	}
	if minioImage == "" {
		return true, nil
	}

	// Get minio snapshot migration status v1.48.0
	kostadmConfig, err := clientset.CoreV1().ConfigMaps(kotsadmNamespace).Get(context.TODO(), "kotsadm-confg", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			// TODO (dans) this behavior needs to change when this feature is opt-out.
			return false, nil
		}
		return false, errors.Wrap(err, "failed to get kotsadm-config map")
	}
	var minioEnabled bool
	if v, ok := kostadmConfig.Data["minio-enabled-snapshots"]; ok {
		minioEnabled, err = strconv.ParseBool(v)
		if err != nil {
			return false, errors.Wrap(err, "failed to parse minio-enabled-snapshots from kotsadm-confg")
		}
		return !minioEnabled, nil
	}

	return false, nil
}
