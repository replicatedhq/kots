package snapshot

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
	registrytypes "github.com/replicatedhq/kotsadm/pkg/registry/types"
	"github.com/replicatedhq/kotsadm/pkg/render"
	"github.com/replicatedhq/kotsadm/pkg/snapshot/types"
	"github.com/replicatedhq/kotsadm/pkg/version"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	velerolabel "github.com/vmware-tanzu/velero/pkg/label"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func CreateBackup(a *app.App) error {
	logger.Debug("creating backup",
		zap.String("appID", a.ID),
		zap.Int64("sequence", a.CurrentSequence))

	archiveDir, err := version.GetAppVersionArchive(a.ID, a.CurrentSequence)
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

	registrySettings, err := getRegistrySettingsForApp(a.ID)
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
		"kots.io/snapshot-trigger": "manual",
		// "kots.io/app-slug":         "", // @areed i don't understand why we need both the slug and id here
		"kots.io/app-id":         a.ID,
		"kots.io/app-sequence":   strconv.FormatInt(a.CurrentSequence, 10),
		"kots.io/snapshot-start": time.Now().UTC().Format(time.RFC3339),
		// "kots.io/cluster-id":       "", // @areed why do we need the cluster id here
	}
	veleroBackup.Spec.IncludedNamespaces = includedNamespaces

	veleroBackup.Spec.StorageLocation = "default"

	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	_, err = veleroClient.Backups(kotsadmVeleroBackendStorageLocation.Namespace).Create(veleroBackup)
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

	veleroBackups, err := veleroClient.Backups(backendStorageLocation.Namespace).List(metav1.ListOptions{})
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
			backup.VolumeSizeHuman = "Many TB"
		}

		if backup.Status != "New" && backup.Status != "InProgress" {
			if !volumeBytesOk || !volumeSuccessCountOk {
				// save computed summary as annotations if snapshot is finished
				vc, vsc, vb, err := getSnapshotVolumeSummary(&veleroBackup)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get volume summary")
				}

				backup.VolumeCount = vc
				backup.VolumeSuccessCount = vsc
				backup.VolumeBytes = vb

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

func getSnapshotVolumeSummary(veleroBackup *velerov1.Backup) (int, int, int64, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return 0, 0, int64(0), errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return 0, 0, int64(0), errors.Wrap(err, "failed to create clientset")
	}

	veleroPodBackupVolumes, err := veleroClient.PodVolumeBackups(veleroBackup.Namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("velero.io/backup-name=%s", velerolabel.GetValidName(veleroBackup.Name)),
	})
	if err != nil {
		return 0, 0, int64(0), errors.Wrap(err, "failed to list pod back up volumes")
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

	return count, success, totalBytes, nil

}

// this is a copy from registry.  so many import cycles to unwind here, todo
func getRegistrySettingsForApp(appID string) (*registrytypes.RegistrySettings, error) {
	db := persistence.MustGetPGSession()
	query := `select registry_hostname, registry_username, registry_password_enc, namespace from app where id = $1`

	row := db.QueryRow(query, appID)

	var registryHostname sql.NullString
	var registryUsername sql.NullString
	var registryPasswordEnc sql.NullString
	var registryNamespace sql.NullString

	if err := row.Scan(&registryHostname, &registryUsername, &registryPasswordEnc, &registryNamespace); err != nil {
		return nil, errors.Wrap(err, "failed to scan registry")
	}

	if !registryHostname.Valid {
		return nil, nil
	}

	registrySettings := registrytypes.RegistrySettings{
		Hostname:    registryHostname.String,
		Username:    registryUsername.String,
		PasswordEnc: registryPasswordEnc.String,
		Namespace:   registryNamespace.String,
	}

	return &registrySettings, nil
}
