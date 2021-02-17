package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"time"

	units "github.com/docker/go-units"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/k8s"
	"github.com/replicatedhq/kots/kotsadm/pkg/kurl"
	"github.com/replicatedhq/kots/kotsadm/pkg/render/helper"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/api/snapshot/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	velerolabel "github.com/vmware-tanzu/velero/pkg/label"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func CreateApplicationBackup(ctx context.Context, a *apptypes.App, isScheduled bool) (*velerov1.Backup, error) {
	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams for app")
	}

	if len(downstreams) == 0 {
		return nil, errors.New("no downstreams found for app")
	}

	parentSequence, err := downstream.GetCurrentParentSequence(a.ID, downstreams[0].ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current downstream parent sequence")
	}
	if parentSequence == -1 {
		return nil, errors.New("app does not have a deployed version")
	}

	logger.Debug("creating backup",
		zap.String("appID", a.ID),
		zap.Int64("sequence", parentSequence))

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(a.ID, parentSequence, archiveDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app version archive")
	}

	kotsadmVeleroBackendStorageLocation, err := FindBackupStoreLocation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load kots kinds from path")
	}

	backupSpec, err := kotsKinds.Marshal("velero.io", "v1", "Backup")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get backup spec from kotskinds")
	}

	if backupSpec == "" {
		return nil, errors.Errorf("application %s does not have a backup spec", a.Slug)
	}

	renderedBackup, err := helper.RenderAppFile(a, nil, []byte(backupSpec), kotsKinds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render backup")
	}
	veleroBackup, err := kotsutil.LoadBackupFromContents(renderedBackup)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load backup from contents")
	}

	appNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	includedNamespaces := []string{appNamespace}
	includedNamespaces = append(includedNamespaces, kotsKinds.KotsApplication.Spec.AdditionalNamespaces...)

	if os.Getenv("KOTSADM_ENV") == "dev" {
		includedNamespaces = append(includedNamespaces, os.Getenv("POD_NAMESPACE"))
	}

	snapshotTrigger := "manual"
	if isScheduled {
		snapshotTrigger = "schedule"
	}

	veleroBackup.Name = ""
	veleroBackup.GenerateName = a.Slug + "-"

	veleroBackup.Namespace = kotsadmVeleroBackendStorageLocation.Namespace
	veleroBackup.Annotations = map[string]string{
		"kots.io/snapshot-trigger":   snapshotTrigger,
		"kots.io/app-id":             a.ID,
		"kots.io/app-sequence":       strconv.FormatInt(parentSequence, 10),
		"kots.io/snapshot-requested": time.Now().UTC().Format(time.RFC3339),
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"kots.io/app-slug": a.Slug,
		},
	}
	if veleroBackup.Spec.LabelSelector != nil {
		labelSelector = mergeLabelSelector(labelSelector, *veleroBackup.Spec.LabelSelector)
	}
	veleroBackup.Spec.LabelSelector = &labelSelector

	veleroBackup.Spec.IncludedNamespaces = includedNamespaces

	veleroBackup.Spec.StorageLocation = "default"

	if a.SnapshotTTL != "" {
		ttlDuration, err := time.ParseDuration(a.SnapshotTTL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse app snapshot ttl value as duration")
		}
		veleroBackup.Spec.TTL = metav1.Duration{
			Duration: ttlDuration,
		}
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	backup, err := veleroClient.Backups(kotsadmVeleroBackendStorageLocation.Namespace).Create(ctx, veleroBackup, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero backup")
	}

	return backup, nil
}

func CreateInstanceBackup(ctx context.Context, cluster *downstreamtypes.Downstream, isScheduled bool) (*velerov1.Backup, error) {
	logger.Debug("creating instance backup")

	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list installed apps")
	}

	kotsadmNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		kotsadmNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	appsSequences := map[string]int64{}
	includedNamespaces := []string{kotsadmNamespace}
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
		},
	}

	for _, a := range apps {
		downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list downstreams for app %s", a.Slug)
		}

		if len(downstreams) == 0 {
			logger.Error(errors.Wrapf(err, "no downstreams found for app %s", a.Slug))
			continue
		}

		parentSequence, err := downstream.GetCurrentParentSequence(a.ID, downstreams[0].ClusterID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get current downstream parent sequence for app %s", a.Slug)
		}
		if parentSequence == -1 {
			// no version is deployed for this app yet
			continue
		}
		appsSequences[a.Slug] = parentSequence

		archiveDir, err := ioutil.TempDir("", "kotsadm")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create temp dir for app %s", a.Slug)
		}
		defer os.RemoveAll(archiveDir)

		err = store.GetStore().GetAppVersionArchive(a.ID, parentSequence, archiveDir)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get app version archive for app %s", a.Slug)
		}

		kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load kots kinds from path")
		}

		backupSpec, err := kotsKinds.Marshal("velero.io", "v1", "Backup")
		if err != nil {
			return nil, errors.Wrap(err, "failed to get backup spec from kotskinds")
		}

		if backupSpec == "" {
			continue
		}

		renderedBackup, err := helper.RenderAppFile(a, nil, []byte(backupSpec), kotsKinds)
		if err != nil {
			return nil, errors.Wrap(err, "failed to render backup")
		}
		veleroBackup, err := kotsutil.LoadBackupFromContents(renderedBackup)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load backup from contents")
		}

		if veleroBackup.Spec.LabelSelector != nil {
			labelSelector = mergeLabelSelector(labelSelector, *veleroBackup.Spec.LabelSelector)
		}

		includedNamespaces = append(includedNamespaces, kotsKinds.KotsApplication.Spec.AdditionalNamespaces...)
	}

	isKurl := kurl.IsKurl()
	if isKurl {
		includedNamespaces = append(includedNamespaces, "kurl")
	}

	if os.Getenv("KOTSADM_ENV") == "dev" {
		includedNamespaces = append(includedNamespaces, os.Getenv("POD_NAMESPACE"))
	}

	kotsadmVeleroBackendStorageLocation, err := FindBackupStoreLocation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}

	kotsadmImage, err := k8s.FindKotsadmImage(kotsadmNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find kotsadm image")
	}

	snapshotTrigger := "manual"
	if isScheduled {
		snapshotTrigger = "schedule"
	}

	// marshal apps sequences map
	b, err := json.Marshal(appsSequences)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal apps sequences")
	}
	marshalledAppsSequences := string(b)

	veleroBackup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:         "",
			GenerateName: "instance-",
			Namespace:    kotsadmVeleroBackendStorageLocation.Namespace,
			Annotations: map[string]string{
				"kots.io/snapshot-trigger":         snapshotTrigger,
				"kots.io/snapshot-requested":       time.Now().UTC().Format(time.RFC3339),
				"kots.io/instance":                 "true",
				"kots.io/kotsadm-image":            kotsadmImage,
				"kots.io/kotsadm-deploy-namespace": kotsadmNamespace,
				"kots.io/apps-sequences":           marshalledAppsSequences,
			},
		},
		Spec: velerov1.BackupSpec{
			StorageLocation:    "default",
			IncludedNamespaces: includedNamespaces,
			LabelSelector:      &labelSelector,
		},
	}

	if isKurl {
		registryHost, _, _, err := kotsutil.GetKurlRegistryCreds()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get kurl registry host")
		}
		veleroBackup.ObjectMeta.Annotations["kots.io/kurl-registry"] = registryHost
	}

	if cluster.SnapshotTTL != "" {
		ttlDuration, err := time.ParseDuration(cluster.SnapshotTTL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse cluster snapshot ttl value as duration")
		}
		veleroBackup.Spec.TTL = metav1.Duration{
			Duration: ttlDuration,
		}
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	backup, err := veleroClient.Backups(kotsadmVeleroBackendStorageLocation.Namespace).Create(ctx, veleroBackup, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero backup")
	}

	return backup, nil
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

	backendStorageLocation, err := FindBackupStoreLocation()
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
				volumeSummary, err := getSnapshotVolumeSummary(context.TODO(), &veleroBackup)
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

func ListInstanceBackups() ([]*types.Backup, error) {
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

	veleroBackups, err := veleroClient.Backups(backendStorageLocation.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list velero backups")
	}

	backups := []*types.Backup{}

	for _, veleroBackup := range veleroBackups.Items {
		// TODO: Enforce version?
		if veleroBackup.Annotations["kots.io/instance"] != "true" {
			continue
		}

		backup := types.Backup{
			Name:         veleroBackup.Name,
			Status:       string(veleroBackup.Status.Phase),
			IncludedApps: make([]types.App, 0),
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

		appAnnotationStr, _ := veleroBackup.Annotations["kots.io/apps-sequences"]
		if len(appAnnotationStr) > 0 {
			var apps map[string]int64
			if err := json.Unmarshal([]byte(appAnnotationStr), &apps); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal apps sequences")
			}
			for slug, sequence := range apps {
				a, err := store.GetStore().GetAppFromSlug(slug)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get app from slug")
				}

				backup.IncludedApps = append(backup.IncludedApps, types.App{
					Slug:       slug,
					Sequence:   sequence,
					Name:       a.Name,
					AppIconURI: a.IconURI,
				})
			}
		}

		if backup.Status != "New" && backup.Status != "InProgress" {
			if !volumeBytesOk || !volumeSuccessCountOk {
				// save computed summary as annotations if snapshot is finished
				volumeSummary, err := getSnapshotVolumeSummary(context.TODO(), &veleroBackup)
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

func getSnapshotVolumeSummary(ctx context.Context, veleroBackup *velerov1.Backup) (*types.VolumeSummary, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	veleroPodBackupVolumes, err := veleroClient.PodVolumeBackups(veleroBackup.Namespace).List(ctx, metav1.ListOptions{
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
	bsl, err := FindBackupStoreLocation()
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

func DeleteBackup(snapshotName string) error {
	bsl, err := FindBackupStoreLocation()
	if err != nil {
		return errors.Wrap(err, "failed to get velero namespace")
	}

	veleroNamespace := bsl.Namespace
	veleroDeleteBackupRequest := &velerov1.DeleteBackupRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapshotName,
			Namespace: veleroNamespace,
		},
		Spec: velerov1.DeleteBackupRequestSpec{
			BackupName: snapshotName,
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

	_, err = veleroClient.DeleteBackupRequests(veleroNamespace).Create(context.TODO(), veleroDeleteBackupRequest, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create delete backup request")
	}

	return nil
}

func HasUnfinishedApplicationBackup(appID string) (bool, error) {
	backups, err := ListBackupsForApp(appID)
	if err != nil {
		return false, errors.Wrap(err, "failed to list backups")
	}

	for _, backup := range backups {
		if backup.Status == "New" || backup.Status == "InProgress" {
			return true, nil
		}
	}

	return false, nil
}

func HasUnfinishedInstanceBackup() (bool, error) {
	backups, err := ListInstanceBackups()
	if err != nil {
		return false, errors.Wrap(err, "failed to list backups")
	}

	for _, backup := range backups {
		if backup.Status == "New" || backup.Status == "InProgress" {
			return true, nil
		}
	}

	return false, nil
}

func GetBackupDetail(ctx context.Context, backupName string) (*types.BackupDetail, error) {
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

	backup, err := veleroClient.Backups(veleroNamespace).Get(ctx, backupName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get backup")
	}

	backupVolumes, err := veleroClient.PodVolumeBackups(veleroNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("velero.io/backup-name=%s", velerolabel.GetValidName(backupName)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list volumes")
	}

	result := &types.BackupDetail{
		Name:       backup.Name,
		Status:     string(backup.Status.Phase),
		Namespaces: backup.Spec.IncludedNamespaces,
		Volumes:    listBackupVolumes(backupVolumes.Items),
	}

	totalBytesDone := int64(0)
	for _, backupVolume := range backupVolumes.Items {
		totalBytesDone += backupVolume.Status.Progress.BytesDone
	}
	result.VolumeSizeHuman = units.HumanSize(float64(totalBytesDone)) // TODO: should this be TotalBytes rather than BytesDone?

	if backup.Status.Phase == velerov1.BackupPhaseCompleted || backup.Status.Phase == velerov1.BackupPhasePartiallyFailed || backup.Status.Phase == velerov1.BackupPhaseFailed {
		errs, warnings, execs, err := downloadBackupLogs(veleroNamespace, backupName)
		result.Errors = errs
		result.Warnings = warnings
		result.Hooks = execs
		if err != nil {
			// do not fail on error
			logger.Error(errors.Wrap(err, "failed to download backup logs"))
		}
	}

	return result, nil
}

func listBackupVolumes(backupVolumes []velerov1.PodVolumeBackup) []types.SnapshotVolume {
	volumes := []types.SnapshotVolume{}
	for _, backupVolume := range backupVolumes {
		v := types.SnapshotVolume{
			Name:           backupVolume.Name,
			SizeBytesHuman: units.HumanSize(float64(backupVolume.Status.Progress.TotalBytes)),
			DoneBytesHuman: units.HumanSize(float64(backupVolume.Status.Progress.BytesDone)),
			Phase:          string(backupVolume.Status.Phase),
		}

		if backupVolume.Status.Progress.TotalBytes > 0 {
			v.CompletionPercent = int(math.Round(float64(backupVolume.Status.Progress.BytesDone/backupVolume.Status.Progress.TotalBytes) * 100))
		}

		if backupVolume.Status.StartTimestamp != nil {
			v.StartedAt = &backupVolume.Status.StartTimestamp.Time

			if backupVolume.Status.Progress.TotalBytes > 0 {
				bytesPerSecond := float64(backupVolume.Status.Progress.BytesDone) / time.Now().Sub(*v.StartedAt).Seconds()
				bytesRemaining := float64(backupVolume.Status.Progress.TotalBytes - backupVolume.Status.Progress.BytesDone)
				v.TimeRemainingSeconds = int(math.Round(bytesRemaining / bytesPerSecond))
			}
		}
		if backupVolume.Status.CompletionTimestamp != nil {
			v.FinishedAt = &backupVolume.Status.CompletionTimestamp.Time
		}

		volumes = append(volumes, v)
	}
	return volumes
}

func downloadBackupLogs(veleroNamespace, backupName string) ([]types.SnapshotError, []types.SnapshotError, []*types.SnapshotHook, error) {
	gzipReader, err := DownloadRequest(veleroNamespace, velerov1.DownloadTargetKindBackupLog, backupName)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to download backup log")
	}
	defer gzipReader.Close()

	errs, warnings, execs, err := parseLogs(gzipReader)
	return errs, warnings, execs, err
}

func mergeLabelSelector(kots metav1.LabelSelector, app metav1.LabelSelector) metav1.LabelSelector {
	for k, v := range app.MatchLabels {
		if _, ok := kots.MatchLabels[k]; ok {
			logger.Errorf("application label %s is already defined, skipping duplicate", k)
			continue
		}
		kots.MatchLabels[k] = v
	}

	kots.MatchExpressions = append(kots.MatchExpressions, app.MatchExpressions...)
	return kots
}
