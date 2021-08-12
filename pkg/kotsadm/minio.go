package kotsadm

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/snapshot"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getMinioYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

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

func ensureMinio(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	size, err := getSize(deployOptions, "minio", resource.MustParse("4Gi"))
	if err != nil {
		return errors.Wrap(err, "failed to get size")
	}

	if err := ensureS3Secret(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure minio secret")
	}

	if err := ensureMinioStatefulset(deployOptions, clientset, size); err != nil {
		return errors.Wrap(err, "failed to ensure minio statefulset")
	}

	if err := ensureMinioService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure minio service")
	}

	return nil
}

func ensureMinioStatefulset(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, size resource.Quantity) error {
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

	_, err = clientset.AppsV1().StatefulSets(deployOptions.Namespace).Update(ctx, existingMinio, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update minio statefulset")
	}

	return nil
}

func ensureMinioService(namespace string, clientset *kubernetes.Clientset) error {
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
	err = snapshot.EnsureLocalVolumeProviderConfigMap(*fsDeployOptions, veleroNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to ensure velero local-volume-provider config map")
	}

	registryOptions, err := GetKotsadmOptionsFromCluster(deployOptions.Namespace, clientset)
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
		}
	}()

	// New bucket name will be assigned during configuration
	storeOptions := snapshot.ConfigureStoreOptions{
		Path:             "/velero", // Data is not moved from the legacy bucket
		FileSystem:       prevFsConfig,
		KotsadmNamespace: deployOptions.Namespace,
		RegistryOptions:  &registryOptions,
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
	currentBackups, err := snapshot.ListInstanceBackups(context.TODO(), snapshot.ListInstanceBackupsOptions{Namespace: deployOptions.Namespace})
	if err != nil {
		return errors.Wrap(err, "failed to list revised backups")
	}

	for _, prevBackup := range previousBackups {
		if !sliceHasBackup(currentBackups, prevBackup.ObjectMeta.Name) {
			return errors.Errorf("failed to find backup %s in the new Velero deployment", prevBackup.Name)
		}
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
