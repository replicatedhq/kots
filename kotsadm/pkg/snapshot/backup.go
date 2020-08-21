package snapshot

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	units "github.com/docker/go-units"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/render"
	"github.com/replicatedhq/kots/kotsadm/pkg/snapshot/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	kotstypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	velerolabel "github.com/vmware-tanzu/velero/pkg/label"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func CreateBackup(a *apptypes.App) error {
	if err := createApplicationBackup(a); err != nil {
		return errors.Wrap(err, "failed to create application backup")
	}

	// uncomment to create disaster recovery snapshots
	// if err := createAdminConsoleBackup(); err != nil {
	// 	return errors.Wrap(err, "failed to create admin console backup")
	// }

	return nil
}

func createApplicationBackup(a *apptypes.App) error {
	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to list downstreams for app")
	}

	if len(downstreams) == 0 {
		return errors.New("no downstreams found for app")
	}

	parentSequence, err := downstream.GetCurrentParentSequence(a.ID, downstreams[0].ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current downstream parent sequence")
	}

	logger.Debug("creating backup",
		zap.String("appID", a.ID),
		zap.Int64("sequence", parentSequence))

	archiveDir, err := version.GetAppVersionArchive(a.ID, parentSequence)
	if err != nil {
		return errors.Wrap(err, "failed to get app version archive")
	}

	kotsadmVeleroBackendStorageLocation, err := findBackupStoreLocation()
	if err != nil {
		return errors.Wrap(err, "failed to find backupstoragelocations")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return errors.Wrap(err, "failed to load kots kinds from path")
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get registry settings for app")
	}

	backupSpec, err := kotsKinds.Marshal("velero.io", "v1", "Backup")
	if err != nil {
		return errors.Wrap(err, "failed to get backup spec from kotskinds")
	}

	renderedBackup, err := render.RenderFile(kotsKinds, registrySettings, []byte(backupSpec))
	if err != nil {
		return errors.Wrap(err, "failed to render backup")
	}
	veleroBackup, err := kotsutil.LoadBackupFromContents(renderedBackup)
	if err != nil {
		return errors.Wrap(err, "failed to load backup from contents")
	}

	appNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	includedNamespaces := []string{appNamespace}
	includedNamespaces = append(includedNamespaces, kotsKinds.KotsApplication.Spec.AdditionalNamespaces...)

	veleroBackup.Name = ""
	veleroBackup.GenerateName = a.Slug + "-"

	veleroBackup.Namespace = kotsadmVeleroBackendStorageLocation.Namespace
	veleroBackup.Annotations = map[string]string{
		"kots.io/snapshot-trigger":   "manual",
		"kots.io/app-id":             a.ID,
		"kots.io/app-sequence":       strconv.FormatInt(parentSequence, 10),
		"kots.io/snapshot-requested": time.Now().UTC().Format(time.RFC3339),
	}
	veleroBackup.Spec.IncludedNamespaces = includedNamespaces

	veleroBackup.Spec.StorageLocation = "default"

	// uncomment for disaster recovery snapshots
	// if veleroBackup.Spec.LabelSelector == nil {
	// 	veleroBackup.Spec.LabelSelector = &metav1.LabelSelector{}
	// }

	// veleroBackup.Spec.LabelSelector.MatchLabels = map[string]string{
	// 	"kots.io/app-slug": a.Slug,
	// }

	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	_, err = veleroClient.Backups(kotsadmVeleroBackendStorageLocation.Namespace).Create(context.TODO(), veleroBackup, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create velero backup")
	}

	return nil
}

func createAdminConsoleBackup() error {
	logger.Debug("creating admin console backup")

	kotsadmVeleroBackendStorageLocation, err := findBackupStoreLocation()
	if err != nil {
		return errors.Wrap(err, "failed to find backupstoragelocations")
	}

	veleroBackup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:         "",
			GenerateName: "kotsadm-",
			Namespace:    kotsadmVeleroBackendStorageLocation.Namespace,
			Annotations: map[string]string{
				"kots.io/snapshot-trigger":   "manual",
				"kots.io/snapshot-requested": time.Now().UTC().Format(time.RFC3339),
				kotstypes.VeleroKey:          kotstypes.VeleroLabelConsoleValue,
			},
		},
		Spec: velerov1.BackupSpec{
			StorageLocation: "default",
			IncludedNamespaces: []string{
				os.Getenv("POD_NAMESPACE"),
			},
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					kotstypes.VeleroKey: kotstypes.VeleroLabelConsoleValue,
				},
			},
		},
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	_, err = veleroClient.Backups(kotsadmVeleroBackendStorageLocation.Namespace).Create(context.TODO(), veleroBackup, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create velero backup")
	}

	return nil
}

func ListBackupsForApp(appID string) ([]*types.Backup, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	backendStorageLocation, err := findBackupStoreLocation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}

	veleroBackups, err := veleroClient.Backups(backendStorageLocation.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list velero backups")
	}

	backups := []*types.Backup{}

	for _, veleroBackup := range veleroBackups.Items {
		if veleroBackup.Annotations["kots.io/app-id"] != appID {
			continue
		}

		backup := types.Backup{
			Name:   veleroBackup.Name,
			Status: string(veleroBackup.Status.Phase),
			AppID:  appID,
		}

		if veleroBackup.Status.StartTimestamp != nil {
			backup.StartedAt = &veleroBackup.Status.StartTimestamp.Time
		}
		if veleroBackup.Status.CompletionTimestamp != nil {
			backup.FinishedAt = &veleroBackup.Status.CompletionTimestamp.Time
		}
		if veleroBackup.Status.Expiration != nil {
			backup.ExpiresAt = &veleroBackup.Status.Expiration.Time
		}
		sequence, ok := veleroBackup.Annotations["kots.io/app-sequence"]
		if ok {
			s, err := strconv.ParseInt(sequence, 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse app sequence")
			}

			backup.Sequence = s
		}
		if backup.Status == "" {
			backup.Status = "New"
		}

		trigger, ok := veleroBackup.Annotations["kots.io/snapshot-trigger"]
		if ok {
			backup.Trigger = trigger
		}

		supportBundleID, ok := veleroBackup.Annotations["kots.io/support-bundle-id"]
		if ok {
			backup.SupportBundleID = supportBundleID
		}

		volumeCount, volumeCountOk := veleroBackup.Annotations["kots.io/snapshot-volume-count"]
		if volumeCountOk {
			i, err := strconv.Atoi(volumeCount)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert volume-count")
			}
			backup.VolumeCount = i
		}

		volumeSuccessCount, volumeSuccessCountOk := veleroBackup.Annotations["kots.io/snapshot-volume-success-count"]
		if volumeSuccessCountOk {
			i, err := strconv.Atoi(volumeSuccessCount)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert volume-success-count")
			}
			backup.VolumeSuccessCount = i
		}

		volumeBytes, volumeBytesOk := veleroBackup.Annotations["kots.io/snapshot-volume-bytes"]
		if volumeBytesOk {
			i, err := strconv.ParseInt(volumeBytes, 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert volume-bytes")
			}
			backup.VolumeBytes = i
			backup.VolumeSizeHuman = units.HumanSize(float64(i))
		}

		if backup.Status != "New" && backup.Status != "InProgress" {
			if !volumeBytesOk || !volumeSuccessCountOk {
				// save computed summary as annotations if snapshot is finished
				volumeSummary, err := getSnapshotVolumeSummary(&veleroBackup)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get volume summary")
				}

				backup.VolumeCount = volumeSummary.VolumeCount
				backup.VolumeSuccessCount = volumeSummary.VolumeSuccessCount
				backup.VolumeBytes = volumeSummary.VolumeBytes
				backup.VolumeSizeHuman = volumeSummary.VolumeSizeHuman

				// This is failing with "the server could not find the requested resource (put backups.velero.io scheduled-1586536961)"
				// veleroBackup.Annotations["kots.io/snapshot-volume-count"] = strconv.Itoa(backup.VolumeCount)
				// veleroBackup.Annotations["kots.io/snapshot-volume-success-count"] = strconv.Itoa(backup.VolumeSuccessCount)
				// veleroBackup.Annotations["kots.io/snapshot-volume-bytes"] = strconv.FormatInt(backup.VolumeBytes, 10)

				// if _, err = veleroClient.Backups(backendStorageLocation.Namespace).UpdateStatus(&veleroBackup); err != nil {
				// 	return nil, errors.Wrap(err, "failed to update velero backup")
				// }
			}
		}

		backups = append(backups, &backup)
	}

	return backups, nil
}

func ListKotsadmBackups() ([]*types.Backup, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	backendStorageLocation, err := findBackupStoreLocation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}

	veleroBackups, err := veleroClient.Backups(backendStorageLocation.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list velero backups")
	}

	backups := []*types.Backup{}

	for _, veleroBackup := range veleroBackups.Items {
		// TODO: Enforce version?
		if veleroBackup.Annotations["kots.io/backup-type"] != "admin-console" {
			continue
		}

		backup := types.Backup{
			Name:   veleroBackup.Name,
			Status: string(veleroBackup.Status.Phase),
			AppID:  "",
		}

		if veleroBackup.Status.StartTimestamp != nil {
			backup.StartedAt = &veleroBackup.Status.StartTimestamp.Time
		}
		if veleroBackup.Status.CompletionTimestamp != nil {
			backup.FinishedAt = &veleroBackup.Status.CompletionTimestamp.Time
		}
		if veleroBackup.Status.Expiration != nil {
			backup.ExpiresAt = &veleroBackup.Status.Expiration.Time
		}
		if backup.Status == "" {
			backup.Status = "New"
		}

		trigger, ok := veleroBackup.Annotations["kots.io/snapshot-trigger"]
		if ok {
			backup.Trigger = trigger
		}

		volumeCount, volumeCountOk := veleroBackup.Annotations["kots.io/snapshot-volume-count"]
		if volumeCountOk {
			i, err := strconv.Atoi(volumeCount)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert volume-count")
			}
			backup.VolumeCount = i
		}

		volumeSuccessCount, volumeSuccessCountOk := veleroBackup.Annotations["kots.io/snapshot-volume-success-count"]
		if volumeSuccessCountOk {
			i, err := strconv.Atoi(volumeSuccessCount)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert volume-success-count")
			}
			backup.VolumeSuccessCount = i
		}

		volumeBytes, volumeBytesOk := veleroBackup.Annotations["kots.io/snapshot-volume-bytes"]
		if volumeBytesOk {
			i, err := strconv.ParseInt(volumeBytes, 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert volume-bytes")
			}
			backup.VolumeBytes = i
			backup.VolumeSizeHuman = units.HumanSize(float64(i))
		}

		if backup.Status != "New" && backup.Status != "InProgress" {
			if !volumeBytesOk || !volumeSuccessCountOk {
				// save computed summary as annotations if snapshot is finished
				volumeSummary, err := getSnapshotVolumeSummary(&veleroBackup)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get volume summary")
				}

				backup.VolumeCount = volumeSummary.VolumeCount
				backup.VolumeSuccessCount = volumeSummary.VolumeSuccessCount
				backup.VolumeBytes = volumeSummary.VolumeBytes
				backup.VolumeSizeHuman = volumeSummary.VolumeSizeHuman
			}
		}

		backups = append(backups, &backup)
	}

	return backups, nil
}

func getSnapshotVolumeSummary(veleroBackup *velerov1.Backup) (*types.VolumeSummary, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	veleroPodBackupVolumes, err := veleroClient.PodVolumeBackups(veleroBackup.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("velero.io/backup-name=%s", velerolabel.GetValidName(veleroBackup.Name)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list pod back up volumes")
	}

	count := 0
	success := 0
	totalBytes := int64(0)

	for _, veleroPodBackupVolume := range veleroPodBackupVolumes.Items {
		count++
		if veleroPodBackupVolume.Status.Phase == velerov1.PodVolumeBackupPhaseCompleted {
			success++
		}

		totalBytes += veleroPodBackupVolume.Status.Progress.BytesDone
	}

	volumeSummary := types.VolumeSummary{
		VolumeCount:        count,
		VolumeSuccessCount: success,
		VolumeBytes:        totalBytes,
		VolumeSizeHuman:    units.HumanSize(float64(totalBytes)),
	}

	return &volumeSummary, nil
}

func GetBackup(snapshotName string) (*velerov1.Backup, error) {
	bsl, err := findBackupStoreLocation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get velero namespace")
	}

	veleroNamespace := bsl.Namespace

	// get the backup
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	backup, err := veleroClient.Backups(veleroNamespace).Get(context.TODO(), snapshotName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get backup")
	}

	return backup, nil
}

func GetKotsadmBackupDetail(backupName string) (*types.BackupDetail, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	backendStorageLocation, err := findBackupStoreLocation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}

	backup, err := veleroClient.Backups(backendStorageLocation.Namespace).Get(context.TODO(), backupName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get backup")
	}

	backupVolumes, err := veleroClient.PodVolumeBackups(backendStorageLocation.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("velero.io/backup-name=%s", backupName),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list volumes")
	}

	result := &types.BackupDetail{
		Name:             backup.Name,
		Status:           string(backup.Status.Phase),
		Namespaces:       backup.Spec.IncludedNamespaces,
		Hooks:            make([]types.SnapshotHook, 0), // TODO:
		Volumes:          make([]types.SnapshotVolume, 0),
		ValidationErrors: backup.Status.ValidationErrors,
		Errors:           make([]types.SnapshotError, 0), // TODO
		Warnings:         make([]types.SnapshotError, 0), // TODO
	}

	totalBytesDone := int64(0)
	totalVolumes := 0
	completedVolumes := 0
	for _, backupVolume := range backupVolumes.Items {
		totalVolumes += 1
		totalBytesDone += backupVolume.Status.Progress.BytesDone
		if backupVolume.Status.Phase == "Completed" {
			completedVolumes += 1
		}

		v := types.SnapshotVolume{
			Name:           backupVolume.Name,
			SizeBytesHuman: units.HumanSize(float64(backupVolume.Status.Progress.TotalBytes)),
			DoneBytesHuman: units.HumanSize(float64(backupVolume.Status.Progress.BytesDone)),
			// TODO: v.CompletionPercent,
			// TODO: v.TimeRemainingSeconds,
			Phase: string(backupVolume.Status.Phase),
		}

		if backupVolume.Status.StartTimestamp != nil {
			v.StartedAt = &backupVolume.Status.StartTimestamp.Time
		}
		if backupVolume.Status.CompletionTimestamp != nil {
			v.FinishedAt = &backupVolume.Status.CompletionTimestamp.Time
		}

		result.Volumes = append(result.Volumes, v)
	}

	result.VolumeSizeHuman = units.HumanSize(float64(totalBytesDone))

	return result, nil
}
