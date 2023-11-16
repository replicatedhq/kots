package kotsadm

import (
	"bytes"
	"context"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/snapshot"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	MinioImageTagDateRegexp = regexp.MustCompile(`RELEASE\.(\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}Z)`)
	// MigrateToMinioXlBeforeTime is the time that the minio version was released that removed the legacy backend
	// that we need to migrate from: https://github.com/minio/minio/releases/tag/RELEASE.2022-10-29T06-21-33Z
	MigrateToMinioXlBeforeTime = time.Date(2022, 10, 29, 6, 21, 33, 0, time.UTC)
)

func getMinioYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	if deployOptions.MigrateToMinioXl {
		var configMap bytes.Buffer
		if err := s.Encode(kotsadmobjects.MinioXlMigrationScriptsConfigMap(deployOptions.Namespace), &configMap); err != nil {
			return nil, errors.Wrap(err, "failed to marshal minio migration configmap")
		}
		docs["minio-xl-migration-configmap.yaml"] = configMap.Bytes()
	}

	size, err := getSize(deployOptions, "minio", resource.MustParse("4Gi"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get size")
	}
	minioSts, err := kotsadmobjects.MinioStatefulset(deployOptions, size)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get minio statefulset definition")
	}
	var statefulset bytes.Buffer
	if err := s.Encode(minioSts, &statefulset); err != nil {
		return nil, errors.Wrap(err, "failed to marshal minio statefulset")
	}
	docs["minio-statefulset.yaml"] = statefulset.Bytes()

	var service bytes.Buffer
	if err := s.Encode(kotsadmobjects.MinioService(deployOptions.Namespace), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal minio service")
	}
	docs["minio-service.yaml"] = service.Bytes()

	return docs, nil
}

func ensureMinio(deployOptions types.DeployOptions, clientset kubernetes.Interface) error {
	size, err := getSize(deployOptions, "minio", resource.MustParse("4Gi"))
	if err != nil {
		return errors.Wrap(err, "failed to get size")
	}

	if err := ensureS3Secret(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure minio secret")
	}

	if deployOptions.MigrateToMinioXl {
		if err := ensureMinioXlMigrationScriptsConfigmap(deployOptions.Namespace, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure minio xl migration scripts configmap")
		}
	}

	if err := ensureMinioStatefulset(deployOptions, clientset, size); err != nil {
		return errors.Wrap(err, "failed to ensure minio statefulset")
	}

	if deployOptions.MigrateToMinioXl {
		if err := ensureMinioXlMigrationStatusConfigmap(deployOptions.Namespace, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure minio xl migration status configmap")
		}
	}

	if err := ensureMinioService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure minio service")
	}

	return nil
}

func ensureMinioStatefulset(deployOptions types.DeployOptions, clientset kubernetes.Interface, size resource.Quantity) error {
	desiredMinio, err := kotsadmobjects.MinioStatefulset(deployOptions, size)
	if err != nil {
		return errors.Wrap(err, "failed to get desired minio statefulset definition")
	}

	ctx := context.TODO()
	existingMinio, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Get(ctx, "kotsadm-minio", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing statefulset")
		}

		_, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Create(ctx, desiredMinio, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create minio statefulset")
		}

		return nil
	}

	if len(existingMinio.Spec.Template.Spec.Containers) != 1 || len(desiredMinio.Spec.Template.Spec.Containers) != 1 {
		return errors.New("minio stateful set cannot be upgraded")
	}

	existingMinio.Spec.Template.Spec.Volumes = desiredMinio.Spec.Template.Spec.DeepCopy().Volumes
	existingMinio.Spec.Template.Spec.Containers[0].Image = desiredMinio.Spec.Template.Spec.Containers[0].Image
	existingMinio.Spec.Template.Spec.Containers[0].VolumeMounts = desiredMinio.Spec.Template.Spec.Containers[0].DeepCopy().VolumeMounts
	existingMinio.Spec.Template.Spec.InitContainers = desiredMinio.Spec.Template.Spec.DeepCopy().InitContainers

	_, err = clientset.AppsV1().StatefulSets(deployOptions.Namespace).Update(ctx, existingMinio, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update minio statefulset")
	}

	return nil
}

func ensureMinioService(namespace string, clientset kubernetes.Interface) error {
	_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), "kotsadm-minio", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), kotsadmobjects.MinioService(namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create service")
		}
	}

	return nil
}

// MigrateExistingMinioFilesystemDeployments excutes a migration of minio snapshots deployment to the local-volume-provider
// plugin and validates the backups are still accessible. The plugin must already be installed on the cluster with velero
// accessible. This function will configure the plugin and update the default backup storage location. If the backup storage location
// cannot be updated or the plugin fails, the
func MigrateExistingMinioFilesystemDeployments(log *logger.CLILogger, deployOptions *types.DeployOptions) error {
	prevFsConfig, err := snapshot.GetCurrentMinioFileSystemConfig(context.TODO(), deployOptions.Namespace)
	if err != nil {
		errors.Wrap(err, "failed to get check for filesystem snapshot")
	}
	if prevFsConfig == nil {
		return nil
	}
	if prevFsConfig.NFS == nil && prevFsConfig.HostPath == nil {
		return nil
	}

	log.Info("Detected existing Minio Snapshot installation. Attempting to migrate.")

	veleroStatus, err := snapshot.DetectVelero(context.TODO(), deployOptions.Namespace)
	if err != nil {
		return errors.Wrap(err, "failed to detect velero for filesystem migration")
	}
	if veleroStatus == nil {
		log.Info("velero is not installed - skipping snapshot migration")
		return nil
	}
	veleroNamespace := veleroStatus.Namespace

	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create velero clientset")
	}

	bsl, err := snapshot.FindBackupStoreLocation(context.TODO(), clientset, veleroClient, deployOptions.Namespace)
	if err != nil {
		return errors.Wrap(err, "failed to find backupstoragelocations")
	}
	if bsl == nil {
		log.Info("Did not detect active Minio snapshot install. Cleaning up.")
		// Minio FS is not longer enabled; clean up and exit
		// Not catching the error here, as this is not required to proceed
		snapshot.DeleteFileSystemMinio(context.TODO(), deployOptions.Namespace)
		return nil
	}

	store, err := snapshot.GetGlobalStore(context.TODO(), deployOptions.Namespace, bsl)
	if err != nil {
		return errors.Wrap(err, "failed to get snapshot store")
	}
	if store == nil || store.FileSystem == nil {
		log.Info("Did not detect active Minio snapshot install. Cleaning up.")
		// Minio FS is not longer enabled; clean up and exit
		// Not catching the error here, as this is not required to proceed
		snapshot.DeleteFileSystemMinio(context.TODO(), deployOptions.Namespace)
		return nil
	}

	if !veleroStatus.ContainsPlugin("local-volume-provider") {
		log.Info("velero local-volume-provider plugin is not installed - migration cannot be performed")
		log.Info("")
		log.Info("---TO INSTALL the local-volume-provider plugin---")
		log.Info("For existing cluster installations, complete the installation instructions here:")
		log.Info("https://github.com/replicatedhq/local-volume-provider")
		log.Info("")
		log.Info("If you're seeing this message on an embedded cluster installation, please contact your vendor for support.")
		log.Info("")
		log.Info("After the plugin has been installed, re-run your upgrade command.")

		return errors.New("velero local-volume-provider plugin is not installed - cannot perform migration")
	}

	previousBsl := bsl.DeepCopy()

	previousBackups, err := snapshot.ListAllBackups(context.TODO(), snapshot.ListInstanceBackupsOptions{Namespace: deployOptions.Namespace})
	if err != nil {
		return errors.Wrap(err, "failed to list existing backups")
	}

	// Add the config map to configure the new plugin
	fsDeployOptions := &snapshot.FileSystemDeployOptions{
		Namespace:        deployOptions.Namespace,
		IsOpenShift:      k8sutil.IsOpenShift(clientset),
		ForceReset:       false,
		FileSystemConfig: *prevFsConfig,
	}
	err = snapshot.EnsureLocalVolumeProviderConfigMaps(*fsDeployOptions, veleroNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to ensure velero local-volume-provider config map")
	}

	registryConfig, err := GetRegistryConfigFromCluster(deployOptions.Namespace, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to get registry options from cluster")
	}

	start := metav1.Now()
	success := false
	defer func() {
		if !success {
			err := snapshot.RevertToMinioFS(context.TODO(), clientset, veleroClient, deployOptions.Namespace, veleroNamespace, previousBsl)
			if err != nil {
				log.Error(errors.Wrap(err, "Could not restore minio backup storage location"))
				return
			}
			log.Info("Minio backup storage location restored")

			err = cleanUpMigrationArtifact(clientset, deployOptions.Namespace)
			if err != nil {
				log.Error(errors.Wrap(err, "Failed to clean up migration artifact"))
			}
		}
	}()

	// New bucket name will be assigned during configuration
	storeOptions := snapshot.ConfigureStoreOptions{
		Path:             "/velero", // Data is not moved from the legacy bucket
		FileSystem:       prevFsConfig,
		KotsadmNamespace: deployOptions.Namespace,
		RegistryConfig:   &registryConfig,
		IsMinioDisabled:  true,
	}
	if _, err = snapshot.ConfigureStore(context.TODO(), storeOptions); err != nil {
		return errors.Wrap(err, "failed to update backup storage location")
	}

	log.ChildActionWithSpinner("Waiting for the snapshot volume to become available.")
	// Wait for the volume to be provisioned (Velero pod will be READY). (Check that the volume is accesible by the Velero pod)
	if err = snapshot.WaitForDefaultBslAvailableAndSynced(context.TODO(), veleroNamespace, start); err != nil {
		log.FinishChildSpinner()
		return errors.Wrap(err, "failed to wait for default backup storage location to be available")
	}
	log.FinishChildSpinner()

	// validate backups
	log.Info("Validating backups have migrated")
	currentBackups, err := snapshot.ListAllBackups(context.TODO(), snapshot.ListInstanceBackupsOptions{Namespace: deployOptions.Namespace})
	if err != nil {
		return errors.Wrap(err, "failed to list revised backups")
	}

	for _, prevBackup := range previousBackups {
		if !sliceHasBackup(currentBackups, prevBackup.ObjectMeta.Name) {
			return errors.Errorf("failed to find backup %s in the new Velero deployment", prevBackup.Name)
		}
	}

	err = createMigrationArtifact(clientset, deployOptions.Namespace)
	if err != nil {
		return errors.Wrap(err, "failed to create migration artifact")
	}

	// Cleanup on success
	success = true
	log.Info("Migrations successful. Cleaning up Minio resources.")

	if err = snapshot.DeleteFileSystemMinio(context.TODO(), deployOptions.Namespace); err != nil {
		return errors.Wrap(err, "failed to cleanup fs minio")
	}

	return nil
}

// sliceHasBackup returns true if a backup with the same name exists in a slice of Velero Backup objects.
func sliceHasBackup(backups []velerov1api.Backup, backupName string) bool {
	for _, backup := range backups {
		if backup.ObjectMeta.Name == backupName {
			return true
		}
	}
	return false
}

func createMigrationArtifact(clientset kubernetes.Interface, namespace string) error {
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   snapshot.SnapshotMigrationArtifactName,
			Labels: kotsadmtypes.GetKotsadmLabels(),
		},
	}

	_, err := clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create config map")
	}

	return nil
}

func cleanUpMigrationArtifact(clientset kubernetes.Interface, namespace string) error {
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), snapshot.SnapshotMigrationArtifactName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to lookup config map")
		}
	} else {
		err = clientset.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), configMap.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to delete config map")
		}
	}
	return nil
}

// IsMinioXlMigrationNeeded checks if the minio statefulset needs to be migrated from FS to XL.
// If the minio statefulset exists, it returns a bool indicating whether a migration is needed and the image of the minio container.
// If the minio statefulset does not exist, it returns false and an empty string.
func IsMinioXlMigrationNeeded(clientset kubernetes.Interface, namespace string) (needsMigration bool, minioImage string, err error) {
	existingMinio, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), "kotsadm-minio", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return false, "", errors.Wrap(err, "failed to get minio statefulset")
		}
		return false, "", nil
	}

	minioImage = existingMinio.Spec.Template.Spec.Containers[0].Image
	needsMigration, err = imageNeedsMinioXlMigration(minioImage)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to check if minio needs migration")
	}

	return needsMigration, minioImage, nil
}

// imageNeedsMinioXlMigration returns true if the minio image is older than the migrate before time (2022-10-29T06-21-33Z).
func imageNeedsMinioXlMigration(minioImage string) (bool, error) {
	existingImageTagDateMatch := MinioImageTagDateRegexp.FindStringSubmatch(minioImage)
	if len(existingImageTagDateMatch) != 2 {
		return false, errors.New("failed to parse existing image tag date")
	}

	existingImageTagDate, err := time.Parse("2006-01-02T15-04-05Z", existingImageTagDateMatch[1])
	if err != nil {
		return false, errors.Wrap(err, "failed to parse existing image tag date")
	}

	return existingImageTagDate.Before(MigrateToMinioXlBeforeTime), nil
}

func ensureAndWaitForMinio(ctx context.Context, deployOptions types.DeployOptions, clientset kubernetes.Interface) (finalErr error) {
	isMinioXlMigrationRunning, err := IsMinioXlMigrationRunning(ctx, clientset, deployOptions.Namespace)
	if err != nil {
		return errors.Wrap(err, "failed to check if minio xl migration is running")
	}

	// if minio xl migration is running, don't update the minio statefulset
	if !isMinioXlMigrationRunning {
		if err := ensureMinio(deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure minio")
		}
	}

	defer func() {
		if finalErr == nil {
			isMinioXlMigrationRunning, err := IsMinioXlMigrationRunning(ctx, clientset, deployOptions.Namespace)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to check if minio xl migration is running"))
			}

			if isMinioXlMigrationRunning {
				if err := MarkMinioXlMigrationComplete(ctx, clientset, deployOptions.Namespace); err != nil {
					logger.Error(errors.Wrap(err, "failed to mark minio xl migration complete"))
				}
			}
		}
	}()

	if err := k8sutil.WaitForStatefulSetReady(ctx, clientset, deployOptions.Namespace, "kotsadm-minio", deployOptions.Timeout); err != nil {
		return errors.Wrap(err, "failed to wait for minio")
	}

	return nil
}

func IsMinioXlMigrationRunning(ctx context.Context, clientset kubernetes.Interface, namespace string) (bool, error) {
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, MinioXlMigrationStatusConfigmapName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return false, errors.Wrapf(err, "failed to get %s configmap", MinioXlMigrationStatusConfigmapName)
		}
		return false, nil
	}

	if cm.Data == nil {
		return false, nil
	}

	return cm.Data["status"] == "running", nil
}

func MarkMinioXlMigrationComplete(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, MinioXlMigrationStatusConfigmapName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to get %s configmap", MinioXlMigrationStatusConfigmapName)
		}
		return nil // no-op
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data["status"] = "complete"

	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update %s configmap", MinioXlMigrationStatusConfigmapName)
	}

	return nil
}
