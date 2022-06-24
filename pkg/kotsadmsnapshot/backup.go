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
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/render/helper"
	kotssnapshot "github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	velerolabel "github.com/vmware-tanzu/velero/pkg/label"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateApplicationBackup(ctx context.Context, a *apptypes.App, isScheduled bool) (*velerov1.Backup, error) {
	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams for app")
	}

	if len(downstreams) == 0 {
		return nil, errors.New("no downstreams found for app")
	}

	parentSequence, err := store.GetStore().GetCurrentParentSequence(a.ID, downstreams[0].ClusterID)
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

	kotsadmNamespace := util.PodNamespace
	kotsadmVeleroBackendStorageLocation, err := kotssnapshot.FindBackupStoreLocation(ctx, kotsadmNamespace)
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

	renderedBackup, err := helper.RenderAppFile(a, nil, []byte(backupSpec), kotsKinds, kotsadmNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render backup")
	}
	veleroBackup, err := kotsutil.LoadBackupFromContents(renderedBackup)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load backup from contents")
	}

	appNamespace := kotsadmNamespace
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	if veleroBackup.Spec.IncludedNamespaces == nil {
		veleroBackup.Spec.IncludedNamespaces = []string{}
	}
	veleroBackup.Spec.IncludedNamespaces = append(veleroBackup.Spec.IncludedNamespaces, appNamespace)
	veleroBackup.Spec.IncludedNamespaces = append(veleroBackup.Spec.IncludedNamespaces, kotsKinds.KotsApplication.Spec.AdditionalNamespaces...)

	snapshotTrigger := "manual"
	if isScheduled {
		snapshotTrigger = "schedule"
	}

	veleroBackup.Name = ""
	veleroBackup.GenerateName = a.Slug + "-"

	veleroBackup.Namespace = kotsadmVeleroBackendStorageLocation.Namespace

	if veleroBackup.Annotations == nil {
		veleroBackup.Annotations = make(map[string]string, 0)
	}
	veleroBackup.Annotations["kots.io/snapshot-trigger"] = snapshotTrigger
	veleroBackup.Annotations["kots.io/app-id"] = a.ID
	veleroBackup.Annotations["kots.io/app-sequence"] = strconv.FormatInt(parentSequence, 10)
	veleroBackup.Annotations["kots.io/snapshot-requested"] = time.Now().UTC().Format(time.RFC3339)

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"kots.io/app-slug": a.Slug,
		},
	}
	if veleroBackup.Spec.LabelSelector != nil {
		labelSelector = mergeLabelSelector(labelSelector, *veleroBackup.Spec.LabelSelector)
	}
	veleroBackup.Spec.LabelSelector = &labelSelector

	includeClusterResources := true
	if veleroBackup.Spec.IncludeClusterResources != nil {
		includeClusterResources = *veleroBackup.Spec.IncludeClusterResources
	}
	veleroBackup.Spec.IncludeClusterResources = &includeClusterResources

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

	cfg, err := k8sutil.GetClusterConfig()
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

	kotsadmNamespace := util.PodNamespace
	appsSequences := map[string]int64{}
	includedNamespaces := []string{kotsadmNamespace}
	excludedNamespaces := []string{}
	backupAnnotations := map[string]string{}
	backupOrderedResources := map[string]string{}
	backupHooks := velerov1.BackupHooks{
		Resources: []velerov1.BackupResourceHookSpec{},
	}
	// non-supported fields that are intentionally left out cuz they might break full snapshots:
	// - includedResources
	// - excludedResources
	// - labelSelector

	appNamespace := kotsadmNamespace
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}
	if appNamespace != kotsadmNamespace {
		includedNamespaces = append(includedNamespaces, appNamespace)
	}

	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list installed apps")
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

		parentSequence, err := store.GetStore().GetCurrentParentSequence(a.ID, downstreams[0].ClusterID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get current downstream parent sequence for app %s", a.Slug)
		}
		if parentSequence == -1 {
			// no version is deployed for this app yet
			continue
		}

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

		appsSequences[a.Slug] = parentSequence

		renderedBackup, err := helper.RenderAppFile(a, nil, []byte(backupSpec), kotsKinds, kotsadmNamespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to render backup")
		}
		veleroBackup, err := kotsutil.LoadBackupFromContents(renderedBackup)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load backup from contents")
		}

		// ** merge app backup info ** //

		// included namespaces
		includedNamespaces = append(includedNamespaces, veleroBackup.Spec.IncludedNamespaces...)
		includedNamespaces = append(includedNamespaces, kotsKinds.KotsApplication.Spec.AdditionalNamespaces...)

		// excluded namespaces
		excludedNamespaces = append(excludedNamespaces, veleroBackup.Spec.ExcludedNamespaces...)

		// annotations
		for k, v := range veleroBackup.Annotations {
			backupAnnotations[k] = v
		}

		// ordered resources
		for k, v := range veleroBackup.Spec.OrderedResources {
			backupOrderedResources[k] = v
		}

		// backup hooks
		backupHooks.Resources = append(backupHooks.Resources, veleroBackup.Spec.Hooks.Resources...)
	}

	isKurl := kurl.IsKurl()
	if isKurl {
		includedNamespaces = append(includedNamespaces, "kurl")
	}

	kotsadmVeleroBackendStorageLocation, err := kotssnapshot.FindBackupStoreLocation(ctx, kotsadmNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s clientset")
	}

	isKotsadmClusterScoped := k8sutil.IsKotsadmClusterScoped(ctx, clientset, kotsadmNamespace)
	if !isKotsadmClusterScoped {
		// in minimal rbac, a kotsadm role and rolebinding will exist in the velero namespace to give kotsadm access to velero.
		// we backup and restore those so that restoring to a new cluster won't require that the user provide those permissions again.
		includedNamespaces = append(includedNamespaces, kotsadmVeleroBackendStorageLocation.Namespace)
	}

	kotsadmImage, err := k8sutil.FindKotsadmImage(kotsadmNamespace)
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

	// add kots annotations
	backupAnnotations["kots.io/snapshot-trigger"] = snapshotTrigger
	backupAnnotations["kots.io/snapshot-requested"] = time.Now().UTC().Format(time.RFC3339)
	backupAnnotations["kots.io/instance"] = "true"
	backupAnnotations["kots.io/kotsadm-image"] = kotsadmImage
	backupAnnotations["kots.io/kotsadm-deploy-namespace"] = kotsadmNamespace
	backupAnnotations["kots.io/apps-sequences"] = marshalledAppsSequences

	includeClusterResources := true
	veleroBackup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:         "",
			GenerateName: "instance-",
			Namespace:    kotsadmVeleroBackendStorageLocation.Namespace,
			Annotations:  backupAnnotations,
		},
		Spec: velerov1.BackupSpec{
			StorageLocation:         "default",
			IncludedNamespaces:      includedNamespaces,
			ExcludedNamespaces:      excludedNamespaces,
			IncludeClusterResources: &includeClusterResources,
			LabelSelector: &metav1.LabelSelector{
				// app label selectors are not supported and we can't merge them since that might exclude kotsadm components
				MatchLabels: map[string]string{
					kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
				},
			},
			OrderedResources: backupOrderedResources,
			Hooks:            backupHooks,
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

	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero clientset")
	}

	backup, err := veleroClient.Backups(kotsadmVeleroBackendStorageLocation.Namespace).Create(ctx, veleroBackup, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero backup")
	}

	return backup, nil
}

func ListBackupsForApp(ctx context.Context, kotsadmNamespace string, appID string) ([]*types.Backup, error) {
	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	backendStorageLocation, err := kotssnapshot.FindBackupStoreLocation(ctx, kotsadmNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}

	veleroBackups, err := veleroClient.Backups(backendStorageLocation.Namespace).List(ctx, metav1.ListOptions{})
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
				volumeSummary, err := getSnapshotVolumeSummary(ctx, &veleroBackup)
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

func ListInstanceBackups(ctx context.Context, kotsadmNamespace string) ([]*types.Backup, error) {
	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	backendStorageLocation, err := kotssnapshot.FindBackupStoreLocation(ctx, kotsadmNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}

	veleroBackups, err := veleroClient.Backups(backendStorageLocation.Namespace).List(ctx, metav1.ListOptions{})
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
					if store.GetStore().IsNotFound(err) {
						// app might not exist in current installation
						continue
					}
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
				volumeSummary, err := getSnapshotVolumeSummary(ctx, &veleroBackup)
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
	cfg, err := k8sutil.GetClusterConfig()
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

func GetBackup(ctx context.Context, kotsadmNamespace string, snapshotName string) (*velerov1.Backup, error) {
	bsl, err := kotssnapshot.FindBackupStoreLocation(ctx, kotsadmNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get velero namespace")
	}

	veleroNamespace := bsl.Namespace

	// get the backup
	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	backup, err := veleroClient.Backups(veleroNamespace).Get(ctx, snapshotName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get backup")
	}

	return backup, nil
}

func DeleteBackup(ctx context.Context, kotsadmNamespace string, snapshotName string) error {
	bsl, err := kotssnapshot.FindBackupStoreLocation(ctx, kotsadmNamespace)
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

	cfg, err := k8sutil.GetClusterConfig()
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

func HasUnfinishedApplicationBackup(ctx context.Context, kotsadmNamespace string, appID string) (bool, error) {
	backups, err := ListBackupsForApp(ctx, kotsadmNamespace, appID)
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

func HasUnfinishedInstanceBackup(ctx context.Context, kotsadmNamespace string) (bool, error) {
	backups, err := ListInstanceBackups(ctx, kotsadmNamespace)
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

func GetBackupDetail(ctx context.Context, kotsadmNamespace string, backupName string) (*types.BackupDetail, error) {
	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	backendStorageLocation, err := kotssnapshot.FindBackupStoreLocation(ctx, kotsadmNamespace)
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
		errs, warnings, execs, err := downloadBackupLogs(ctx, veleroNamespace, backupName)
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

func downloadBackupLogs(ctx context.Context, veleroNamespace, backupName string) ([]types.SnapshotError, []types.SnapshotError, []*types.SnapshotHook, error) {
	gzipReader, err := DownloadRequest(ctx, veleroNamespace, velerov1.DownloadTargetKindBackupLog, backupName)
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
