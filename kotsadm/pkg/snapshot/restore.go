package snapshot

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	units "github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/snapshot/types"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	velerolabel "github.com/vmware-tanzu/velero/pkg/label"
	"go.uber.org/zap"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func GetRestore(snapshotName string) (*velerov1.Restore, error) {
	bsl, err := FindBackupStoreLocation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get velero namespace")
	}

	veleroNamespace := bsl.Namespace

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	restore, err := veleroClient.Restores(veleroNamespace).Get(context.TODO(), snapshotName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get restore")
	}

	return restore, nil
}

func CreateRestore(snapshotName string) error {
	// Reference https://github.com/vmware-tanzu/velero/blob/42b612645863c2b3e451b447f9bf798295dd7dba/pkg/cmd/cli/restore/create.go#L222

	logger.Debug("creating restore",
		zap.String("snapshotName", snapshotName))

	bsl, err := FindBackupStoreLocation()
	if err != nil {
		return errors.Wrap(err, "failed to get velero namespace")
	}

	veleroNamespace := bsl.Namespace

	// get the backup
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	_, err = veleroClient.Backups(veleroNamespace).Get(context.TODO(), snapshotName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to find backup")
	}

	trueVal := true
	restore := &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: veleroNamespace,
			Name:      snapshotName, // restore name same as snapshot name
		},
		Spec: velerov1.RestoreSpec{
			BackupName:              snapshotName,
			RestorePVs:              &trueVal,
			IncludeClusterResources: &trueVal,
		},
	}

	_, err = veleroClient.Restores(veleroNamespace).Create(context.TODO(), restore, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create restore")
	}

	return nil
}

func DeleteRestore(snapshotName string) error {
	bsl, err := FindBackupStoreLocation()
	if err != nil {
		return errors.Wrap(err, "failed to get velero namespace")
	}

	veleroNamespace := bsl.Namespace

	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	err = veleroClient.Restores(veleroNamespace).Delete(context.TODO(), snapshotName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return errors.Wrapf(err, "failed to delete restore %s", snapshotName)
	}

	return nil
}

func GetKotsadmRestoreDetail(ctx context.Context, restoreName string) (*types.RestoreDetail, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	backendStorageLocation, err := FindBackupStoreLocation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}

	veleroNamespace := backendStorageLocation.Namespace

	restore, err := veleroClient.Restores(veleroNamespace).Get(ctx, restoreName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get restore")
	}

	restoreVolumes, err := veleroClient.PodVolumeRestores(veleroNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("velero.io/restore-name=%s", velerolabel.GetValidName(restore.Name)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list volumes")
	}

	result := &types.RestoreDetail{
		Name:     restore.Name,
		Phase:    string(restore.Status.Phase),
		Volumes:  listRestoreVolumes(restoreVolumes.Items),
		Errors:   make([]types.SnapshotError, 0),
		Warnings: make([]types.SnapshotError, 0),
	}

	if restore.Status.Phase == velerov1.RestorePhaseCompleted || restore.Status.Phase == velerov1.RestorePhasePartiallyFailed || restore.Status.Phase == velerov1.RestorePhaseFailed {
		warnings, errs, err := DownloadRestoreResults(veleroNamespace, restore.Name)
		if err != nil {
			// do not fail on error
			logger.Error(errors.Wrap(err, "failed to download restore results"))
		}

		result.Warnings = warnings
		result.Errors = errs
	}

	return result, nil
}

func listRestoreVolumes(restoreVolumes []velerov1.PodVolumeRestore) []types.RestoreVolume {
	volumes := []types.RestoreVolume{}
	for _, restoreVolume := range restoreVolumes {
		v := types.RestoreVolume{
			Name:           restoreVolume.Name,
			PodName:        restoreVolume.Spec.Pod.Name,
			PodNamespace:   restoreVolume.Spec.Pod.Namespace,
			PodVolumeName:  restoreVolume.Spec.Volume,
			SizeBytesHuman: units.HumanSize(float64(restoreVolume.Status.Progress.TotalBytes)),
			DoneBytesHuman: units.HumanSize(float64(restoreVolume.Status.Progress.BytesDone)),
			Phase:          string(restoreVolume.Status.Phase),
		}

		if restoreVolume.Status.Progress.TotalBytes > 0 {
			v.CompletionPercent = int(math.Round(float64(restoreVolume.Status.Progress.BytesDone/restoreVolume.Status.Progress.TotalBytes) * 100))
		}

		if restoreVolume.Status.StartTimestamp != nil {
			v.StartedAt = &restoreVolume.Status.StartTimestamp.Time

			if restoreVolume.Status.Progress.TotalBytes > 0 {
				if restoreVolume.Status.Progress.BytesDone > 0 {
					bytesPerSecond := float64(restoreVolume.Status.Progress.BytesDone) / time.Now().Sub(*v.StartedAt).Seconds()
					bytesRemaining := float64(restoreVolume.Status.Progress.TotalBytes - restoreVolume.Status.Progress.BytesDone)
					v.RemainingSecondsExist = true
					v.TimeRemainingSeconds = int(math.Round(bytesRemaining / bytesPerSecond))
				} else {
					v.RemainingSecondsExist = false
					v.TimeRemainingSeconds = 0
				}
			}
		}
		if restoreVolume.Status.CompletionTimestamp != nil {
			v.FinishedAt = &restoreVolume.Status.CompletionTimestamp.Time
		}

		volumes = append(volumes, v)
	}
	return volumes
}
