package kotsadm

import (
	"bytes"
	"context"
	"fmt"
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
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
)

var (
	MinioImageTagDateRegexp = regexp.MustCompile(`RELEASE\.(\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}Z)`)
	// MigrateMinioXlBeforeTime is the time that the minio version was released that removed the legacy backend
	// that we need to migrate from: https://github.com/minio/minio/releases/tag/RELEASE.2022-10-29T06-21-33Z
	MigrateMinioXlBeforeTime = time.Date(2022, 10, 29, 6, 21, 33, 0, time.UTC)
)

func getMinioYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	size, err := getSize(deployOptions, "minio", resource.MustParse("4Gi"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get size")
	}
	minioSts, err := kotsadmobjects.MinioStatefulset(deployOptions, size, "kotsadm-minio")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get minio statefulset definition")
	}
	var statefulset bytes.Buffer
	if err := s.Encode(minioSts, &statefulset); err != nil {
		return nil, errors.Wrap(err, "failed to marshal minio statefulset")
	}
	docs["minio-statefulset.yaml"] = statefulset.Bytes()

	var service bytes.Buffer
	if err := s.Encode(kotsadmobjects.MinioService(deployOptions.Namespace, "kotsadm-minio"), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal minio service")
	}
	docs["minio-service.yaml"] = service.Bytes()

	return docs, nil
}

func ensureMinio(deployOptions types.DeployOptions, clientset kubernetes.Interface, name string) error {
	size, err := getSize(deployOptions, "minio", resource.MustParse("4Gi"))
	if err != nil {
		return errors.Wrap(err, "failed to get size")
	}

	if err := ensureS3Secret(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure minio secret")
	}

	if err := ensureMinioStatefulset(deployOptions, clientset, size, name); err != nil {
		return errors.Wrap(err, "failed to ensure minio statefulset")
	}

	if err := ensureMinioService(deployOptions.Namespace, clientset, name); err != nil {
		return errors.Wrap(err, "failed to ensure minio service")
	}

	return nil
}

func ensureMinioStatefulset(deployOptions types.DeployOptions, clientset kubernetes.Interface, size resource.Quantity, name string) error {
	desiredMinio, err := kotsadmobjects.MinioStatefulset(deployOptions, size, name)
	if err != nil {
		return errors.Wrap(err, "failed to get desired minio statefulset definition")
	}

	ctx := context.TODO()
	existingMinio, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Get(ctx, name, metav1.GetOptions{})
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

	_, err = clientset.AppsV1().StatefulSets(deployOptions.Namespace).Update(ctx, existingMinio, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update minio statefulset")
	}

	return nil
}

func ensureMinioService(namespace string, clientset kubernetes.Interface, name string) error {
	_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), kotsadmobjects.MinioService(namespace, name), metav1.CreateOptions{})
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

	bsl, err := snapshot.FindBackupStoreLocation(context.TODO(), deployOptions.Namespace)
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
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}
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
			err := snapshot.RevertToMinioFS(context.TODO(), deployOptions.Namespace, veleroNamespace, previousBsl)
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
func IsMinioXlMigrationNeeded(clientset kubernetes.Interface, namespace string) (bool, error) {
	existingMinio, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), "kotsadm-minio", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return false, errors.Wrap(err, "failed to get minio statefulset")
		}
		return false, nil
	}

	needsMigration, err := imageNeedsMinioXlMigration(existingMinio.Spec.Template.Spec.Containers[0].Image)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if minio needs migration")
	}
	return needsMigration, nil
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

	return existingImageTagDate.Before(MigrateMinioXlBeforeTime), nil
}

func MigrateExistingMinioBackend(deployOptions types.DeployOptions, clientset kubernetes.Interface, log *logger.CLILogger, namespace string) (err error) {
	// if the migration fails, we need to restore the existing minio resources

	needsMigration, err := IsMinioXlMigrationNeeded(clientset, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to check if minio needs migration")
	}

	if !needsMigration {
		return nil
	}

	log.Info("Detected minio with legacy backend. Attempting to migrate.")

	defer func() {
		if err != nil {
			log.Info("Minio backend migration failed. Restoring existing Minio resources.")
			err := restoreExistingMinio(clientset, namespace)
			if err != nil {
				log.Error(errors.Wrap(err, "failed to restore existing minio"))
			}
		}
	}()

	// get the pods that are part of the old minio statefulset
	oldMinioPods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", "kotsadm-minio"),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list existing minio pods")
	}

	if len(oldMinioPods.Items) == 0 {
		return errors.New("no existing minio pods found")
	} else if len(oldMinioPods.Items) > 1 {
		return errors.New("more than one existing minio pod found")
	}

	oldMinioPod := oldMinioPods.Items[0]

	// get the pvc for this pod
	oldMinioPVC, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), oldMinioPod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get existing minio pvc")
	}

	// // get the pv for the old minio instance
	// oldMinioPV, err := clientset.CoreV1().PersistentVolumes().Get(context.TODO(), oldMinioPVC.Spec.VolumeName, metav1.GetOptions{})
	// if err != nil {
	// 	return errors.Wrap(err, "failed to get existing minio pv")
	// }

	// // set the reclaim policy to retain
	// oldMinioPV.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
	// _, err = clientset.CoreV1().PersistentVolumes().Update(context.TODO(), oldMinioPV, metav1.UpdateOptions{})
	// if err != nil {
	// 	return errors.Wrap(err, "failed to update existing minio pv")
	// }

	if err := ensureMinioXlMigrationScriptsConfigmap(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure minio scripts configmap")
	}

	if err := ensureMinio(deployOptions, clientset, "kotsadm-minio-xl-migration"); err != nil {
		return errors.Wrap(err, "failed to create new minio instance")
	}

	// wait for the kotsadm-minio-xl-migration statefulset to be ready
	timeout := time.Now().Add(deployOptions.Timeout)
	for {
		if time.Now().After(timeout) {
			return errors.New("timed out waiting for new minio instance to be ready")
		}

		// check if the new minio instance is ready
		newMinioStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), "kotsadm-minio-xl-migration", metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get new minio statefulset")
		}

		if newMinioStatefulSet.Status.ReadyReplicas == 1 {
			break
		}

		time.Sleep(5 * time.Second)
	}

	// get the pods that are part of the new minio statefulset
	newMinioPods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", "kotsadm-minio-xl-migration"),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list new minio pods")
	}

	if len(newMinioPods.Items) == 0 {
		return errors.New("no new minio pods found")
	} else if len(newMinioPods.Items) > 1 {
		return errors.New("more than one new minio pod found")
	}

	newMinioPod := newMinioPods.Items[0]

	// get the pvc for this pod
	newMinioPVC, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), newMinioPod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get new minio pvc")
	}

	// wait for the new pvc to be bound or timeout
	timeout = time.Now().Add(deployOptions.Timeout)
	for {
		if time.Now().After(timeout) {
			return errors.New("timed out waiting for new minio pvc to be bound")
		}

		newMinioPVC, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), newMinioPVC.Name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get new minio pvc")
		}

		if newMinioPVC.Status.Phase == corev1.ClaimBound {
			break
		}

		time.Sleep(time.Second)
	}

	// // get the pv for the new minio instance
	// newMinioPV, err := clientset.CoreV1().PersistentVolumes().Get(context.TODO(), newMinioPVC.Spec.VolumeName, metav1.GetOptions{})
	// if err != nil {
	// 	return errors.Wrap(err, "failed to get new minio pv")
	// }

	// // set the reclaim policy to retain
	// newMinioPV.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
	// _, err = clientset.CoreV1().PersistentVolumes().Update(context.TODO(), newMinioPV, metav1.UpdateOptions{})
	// if err != nil {
	// 	return errors.Wrap(err, "failed to update new minio pv")
	// }

	// 3. copy the data from the old minio instance to the new minio instance
	// this is done by creating a job that runs through the steps documented here: https://min.io/docs/minio/linux/operations/install-deploy-manage/migrate-fs-gateway.html
	job, err := clientset.BatchV1().Jobs(namespace).Create(context.TODO(), kotsadmobjects.MinioXlMigrationJob(deployOptions, namespace), metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create minio xl migration job")
	}

	// wait for the job to complete or timeout
	timeout = time.Now().Add(deployOptions.Timeout)
	for {
		if time.Now().After(timeout) {
			return errors.New("timed out waiting for minio xl migration job to complete")
		}

		job, err = clientset.BatchV1().Jobs(namespace).Get(context.TODO(), job.Name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get minio xl migration job")
		}

		if job.Status.Succeeded > 0 {
			break
		} else if job.Status.Failed > 0 {
			return errors.New("minio xl migration job failed")
		}

		time.Sleep(time.Second)
	}

	log.Info("Minio xl migration job completed successfully") // TODO: remove

	// delete the job
	err = clientset.BatchV1().Jobs(namespace).Delete(context.TODO(), job.Name, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to delete minio xl migration job")
	}

	log.Info("scaling down old minio instance...")

	// 4. scale down the old minio instance (and the new?)
	oldMinioStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), "kotsadm-minio", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get old minio statefulset")
	}

	oldMinioStatefulSet.Spec.Replicas = pointer.Int32Ptr(0)
	_, err = clientset.AppsV1().StatefulSets(namespace).Update(context.TODO(), oldMinioStatefulSet, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update old minio statefulset")
	}

	log.Info("scaling down new minio instance...")

	newMinioStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), "kotsadm-minio-xl-migration", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get new minio statefulset")
	}

	newMinioStatefulSet.Spec.Replicas = pointer.Int32Ptr(0)
	_, err = clientset.AppsV1().StatefulSets(namespace).Update(context.TODO(), newMinioStatefulSet, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update new minio statefulset")
	}

	// 5. swap the pvcs

	// // get the old minio pvc (TODO: Is this necessary, or can we use the one from above)
	// oldMinioPVC, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), oldPVCName, metav1.GetOptions{})
	// if err != nil {
	// 	return errors.Wrap(err, "failed to get old minio pvc")
	// }

	log.Info("swapping pvc...")

	log.Info("setting old minio pvc to retain...")

	// get the old minio pv
	oldMinioPV, err := clientset.CoreV1().PersistentVolumes().Get(context.TODO(), oldMinioPVC.Spec.VolumeName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get old minio pv")
	}

	// set the reclaim policy to retain
	oldMinioPV.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
	_, err = clientset.CoreV1().PersistentVolumes().Update(context.TODO(), oldMinioPV, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update old minio pv")
	}

	log.Info("setting new minio pvc to retain...")

	// get the new minio pv
	newMinioPV, err := clientset.CoreV1().PersistentVolumes().Get(context.TODO(), newMinioPVC.Spec.VolumeName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get new minio pv")
	}

	// set the reclaim policy to retain
	newMinioPV.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
	_, err = clientset.CoreV1().PersistentVolumes().Update(context.TODO(), newMinioPV, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update new minio pv")
	}

	log.Info("deleting old minio pvc...")

	// delete the old minio pvc
	err = clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), oldMinioPVC.Name, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to delete old minio pvc")
	}

	log.Info("deleting new minio pvc...")

	// delete the new minio pvc
	err = clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), newMinioPVC.Name, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to delete new minio pvc")
	}

	log.Info("creating old minio pvc with new minio pv...")

	// wait for old minio pvc to be deleted
	log.Info("waiting for old minio pvc to be deleted...")
	for {
		_, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), oldMinioPVC.Name, metav1.GetOptions{})
		if err != nil {
			if kuberneteserrors.IsNotFound(err) {
				break
			}
			return errors.Wrap(err, "failed to get old minio pvc")
		}
		time.Sleep(time.Second)
	}
	log.Info("old minio pvc deleted")

	// create the a new pvc for the old minio statefulset that points to the new minio pv
	newPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oldMinioPVC.Name,
			Namespace: oldMinioPVC.Namespace,
			Labels:    oldMinioPVC.Labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      oldMinioPVC.Spec.AccessModes,
			Resources:        oldMinioPVC.Spec.Resources,
			StorageClassName: oldMinioPVC.Spec.StorageClassName,
			VolumeName:       newMinioPV.Name,
		},
	}

	// set the claim ref to the new minio pv
	newMinioPV, err = clientset.CoreV1().PersistentVolumes().Get(context.TODO(), newMinioPV.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get new minio pv")
	}

	newMinioPV.Spec.ClaimRef = &corev1.ObjectReference{
		Kind:       "PersistentVolumeClaim",
		APIVersion: "v1",
		Namespace:  namespace,
		Name:       newPVC.Name,
		UID:        newPVC.UID,
	}

	_, err = clientset.CoreV1().PersistentVolumes().Update(context.TODO(), newMinioPV, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update new minio pv")
	}

	// create the new pvc
	_, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), newPVC, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create new minio pvc")
	}

	log.Info("waiting for new minio pvc to be bound...")
	for {
		newMinioPVC, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), newPVC.Name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get new minio pvc")
		}
		if newMinioPVC.Status.Phase == corev1.ClaimBound {
			break
		}
		time.Sleep(time.Second)
	}

	log.Info("scaling up old minio instance...")

	// get the latest version of the old minio statefulset
	oldMinioStatefulSet, err = clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), oldMinioStatefulSet.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get old minio statefulset")
	}

	oldMinioStatefulSet.Spec.Replicas = pointer.Int32Ptr(1)
	// TODO: patch the image here to prevent a crashloop?
	oldMinioStatefulSet.Spec.Template.Spec.Containers[0].Image = kotsadmobjects.GetAdminConsoleImage(deployOptions, "minio")
	_, err = clientset.AppsV1().StatefulSets(namespace).Update(context.TODO(), oldMinioStatefulSet, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update old minio statefulset")
	}

	log.Info("waiting for old minio instance to be ready...")
	for {
		oldMinioStatefulSet, err = clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), oldMinioStatefulSet.Name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get old minio statefulset")
		}

		if oldMinioStatefulSet.Status.ReadyReplicas == 1 {
			break
		}

		time.Sleep(time.Second)
	}

	log.Info("deleting new minio instance...")

	// 8. delete the new minio instance
	err = clientset.AppsV1().StatefulSets(namespace).Delete(context.TODO(), newMinioStatefulSet.Name, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to delete new minio statefulset")
	}

	// get the latest version of the old pv
	oldMinioPV, err = clientset.CoreV1().PersistentVolumes().Get(context.TODO(), oldMinioPV.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get old minio pv")
	}

	// annotate the old minio pv with a label that indicates in can be deleted
	oldMinioPV.ObjectMeta.Annotations["kots.io/minio-fs-volume"] = "true"
	_, err = clientset.CoreV1().PersistentVolumes().Update(context.TODO(), oldMinioPV, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update old minio pv")
	}

	log.Info("Minio xl migration completed successfully")

	return nil
}

func restoreExistingMinio(clientset kubernetes.Interface, namespace string) error {
	// TODO: restore the existing minio resources - aka undo the migration
	return nil
}
