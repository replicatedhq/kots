package snapshot

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	units "github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	"github.com/replicatedhq/kots/pkg/logger"
	kotssnapshot "github.com/replicatedhq/kots/pkg/snapshot"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	velerolabel "github.com/vmware-tanzu/velero/pkg/label"
	"go.uber.org/zap"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
)

func GetRestore(ctx context.Context, kotsadmNamespace string, snapshotName string) (*velerov1.Restore, error) {
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

	bsl, err := kotssnapshot.FindBackupStoreLocation(ctx, clientset, veleroClient, kotsadmNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get velero namespace")
	}
	if bsl == nil {
		return nil, errors.New("no backup store location found")
	}

	veleroNamespace := bsl.Namespace

	restore, err := veleroClient.Restores(veleroNamespace).Get(ctx, snapshotName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get restore")
	}

	return restore, nil
}

func CreateApplicationRestore(ctx context.Context, kotsadmNamespace string, snapshotName string, appSlug string) error {
	// Reference https://github.com/vmware-tanzu/velero/blob/42b612645863c2b3e451b447f9bf798295dd7dba/pkg/cmd/cli/restore/create.go#L222

	logger.Debug("creating restore",
		zap.String("snapshotName", snapshotName))

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

	bsl, err := kotssnapshot.FindBackupStoreLocation(ctx, clientset, veleroClient, kotsadmNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to get velero namespace")
	}
	if bsl == nil {
		return errors.New("no backup store location found")
	}

	veleroNamespace := bsl.Namespace

	// get the backup
	backup, err := veleroClient.Backups(veleroNamespace).Get(ctx, snapshotName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to find backup")
	}

	restore := &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: veleroNamespace,
			Name:      snapshotName, // restore name same as snapshot name
		},
		Spec: velerov1.RestoreSpec{
			BackupName:              snapshotName,
			RestorePVs:              pointer.Bool(true),
			IncludeClusterResources: pointer.Bool(true),
		},
	}

	if backup.Annotations["kots.io/instance"] == "true" {
		// only restore app-specific objects
		restore.ObjectMeta.Name = fmt.Sprintf("%s.%s", snapshotName, appSlug)
		restore.ObjectMeta.Annotations = map[string]string{
			"kots.io/instance": "true",
		}
		restore.Spec.LabelSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"kots.io/app-slug": appSlug,
			},
		}
	}

	_, err = veleroClient.Restores(veleroNamespace).Create(ctx, restore, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create restore")
	}

	// TODO(improveddr)
	// create the kotsKinds.Restore included with the yaml from the vendor
	// Add the EC annotation

	return nil
}

func DeleteRestore(ctx context.Context, kotsadmNamespace string, snapshotName string) error {
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

	bsl, err := kotssnapshot.FindBackupStoreLocation(ctx, clientset, veleroClient, kotsadmNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to get velero namespace")
	}

	veleroNamespace := bsl.Namespace

	err = veleroClient.Restores(veleroNamespace).Delete(ctx, snapshotName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return errors.Wrapf(err, "failed to delete restore %s", snapshotName)
	}

	return nil
}

func GetRestoreDetails(ctx context.Context, kotsadmNamespace string, restoreName string) (*types.RestoreDetail, error) {
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

	backendStorageLocation, err := kotssnapshot.FindBackupStoreLocation(ctx, clientset, veleroClient, kotsadmNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}
	if backendStorageLocation == nil {
		return nil, errors.New("no backup store location found")
	}

	veleroNamespace := backendStorageLocation.Namespace

	restore, err := veleroClient.Restores(veleroNamespace).Get(ctx, restoreName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get restore")
	}

	restoreVolumes, err := veleroClient.PodVolumeRestores(veleroNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("velero.io/restore-name=%s", velerolabel.GetValidName(restore.Name)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list volumes")
	}

	result := &types.RestoreDetail{
		Name:     restore.Name,
		Phase:    restore.Status.Phase,
		Volumes:  listRestoreVolumes(restoreVolumes.Items),
		Errors:   make([]types.SnapshotError, 0),
		Warnings: make([]types.SnapshotError, 0),
	}

	if restore.Status.Phase == velerov1.RestorePhaseCompleted || restore.Status.Phase == velerov1.RestorePhasePartiallyFailed || restore.Status.Phase == velerov1.RestorePhaseFailed {
		warnings, errs, err := DownloadRestoreResults(ctx, veleroNamespace, restore.Name)
		if err != nil {
			// do not fail on error
			logger.Error(errors.Wrap(err, "failed to download restore results"))
		}

		result.Errors = errs

		filtered, err := filterWarnings(restore, warnings, &filterGetter{})
		if err != nil {
			logger.Infof("failed to filter warnings: %v", err)
			result.Warnings = warnings
		} else {
			result.Warnings = filtered
		}
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
