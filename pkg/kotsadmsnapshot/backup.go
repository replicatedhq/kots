package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	units "github.com/docker/go-units"
	"github.com/pkg/errors"
	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	InstanceBackupNameAnnotation      = "replicated.com/backup-name"
	InstanceBackupTypeAnnotation      = "replicated.com/backup-type"
	InstanceBackupsExpectedAnnotation = "replicated.com/backups-expected"

	InstanceBackupTypeKotsadm  = "kotsadm"
	InstanceBackupTypeApp      = "app"
	InstanceBackupTypeCombined = "combined"
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

	archiveDir, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(a.ID, parentSequence, archiveDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app version archive")
	}

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

	kotsadmNamespace := util.PodNamespace
	kotsadmVeleroBackendStorageLocation, err := kotssnapshot.FindBackupStoreLocation(ctx, clientset, veleroClient, kotsadmNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}

	if kotsadmVeleroBackendStorageLocation == nil {
		return nil, errors.New("no backup store location found")
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
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

	includedNamespaces := []string{}
	includedNamespaces = append(includedNamespaces, appNamespace)
	includedNamespaces = append(includedNamespaces, veleroBackup.Spec.IncludedNamespaces...)
	includedNamespaces = append(includedNamespaces, kotsKinds.KotsApplication.Spec.AdditionalNamespaces...)

	veleroBackup.Spec.IncludedNamespaces = prepareIncludedNamespaces(includedNamespaces)

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

	err = excludeShutdownPodsFromBackup(ctx, clientset, veleroBackup)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to exclude shutdown pods from backup"))
	}

	backup, err := veleroClient.Backups(kotsadmVeleroBackendStorageLocation.Namespace).Create(ctx, veleroBackup, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero backup")
	}

	return backup, nil
}

type instanceBackupMetadata struct {
	backupName                     string
	backupReqestedAt               time.Time
	kotsadmNamespace               string
	backupStorageLocationNamespace string
	apps                           map[string]appInstanceBackupMetadata
	isScheduled                    bool
	snapshotTTL                    time.Duration
	ec                             *ecInstanceBackupMetadata
}

type appInstanceBackupMetadata struct {
	app            *apptypes.App
	kotsKinds      *kotsutil.KotsKinds
	parentSequence int64
}

type ecInstanceBackupMetadata struct {
	installation         *embeddedclusterv1beta1.Installation
	seaweedFSS3ServiceIP string
}

func CreateInstanceBackup(ctx context.Context, cluster *downstreamtypes.Downstream, isScheduled bool) (string, error) {
	logger.Info("Creating instance backup")

	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to get cluster config")
	}

	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to create clientset")
	}

	ctrlClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get kubeclient: %w", err)
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to create velero clientset")
	}

	metadata, err := getInstanceBackupMetadata(ctx, k8sClient, ctrlClient, veleroClient, cluster, isScheduled)
	if err != nil {
		return "", errors.Wrap(err, "failed to get instance backup metadata")
	}

	appVeleroBackup, err := getAppInstanceBackupSpec(ctx, k8sClient, metadata)
	if err != nil {
		return "", errors.Wrap(err, "failed to get app instance backup spec")
	}

	veleroBackup, err := getInstanceBackupSpec(ctx, k8sClient, metadata, appVeleroBackup != nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to get instance backup specs")
	}

	err = excludeShutdownPodsFromBackup(ctx, k8sClient, veleroBackup)
	if err != nil {
		logger.Errorf("Failed to exclude shutdown pods from backup: %v", err)
	}

	if appVeleroBackup != nil {
		err = excludeShutdownPodsFromBackup(ctx, k8sClient, appVeleroBackup)
		if err != nil {
			logger.Errorf("Failed to exclude shutdown pods from application backup: %v", err)
		}
	}

	logger.Info("Creating instance backup CR", zap.String("name", veleroBackup.Name))
	_, err = veleroClient.Backups(metadata.backupStorageLocationNamespace).Create(ctx, veleroBackup, metav1.CreateOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to create velero backup")
	}

	if appVeleroBackup != nil {
		logger.Info("Creating instance app backup CR", zap.String("name", appVeleroBackup.Name))
		_, err := veleroClient.Backups(metadata.backupStorageLocationNamespace).Create(ctx, appVeleroBackup, metav1.CreateOptions{})
		if err != nil {
			return "", errors.Wrap(err, "failed to create application velero backup")
		}
	}

	return metadata.backupName, nil
}

func GetBackupName(veleroBackup velerov1.Backup) string {
	if val, ok := veleroBackup.GetAnnotations()[InstanceBackupNameAnnotation]; ok {
		return val
	}
	return veleroBackup.GetName()
}

func GetInstanceBackupType(veleroBackup velerov1.Backup) string {
	if val, ok := veleroBackup.GetAnnotations()[InstanceBackupTypeAnnotation]; ok {
		return val
	}
	return ""
}

func GetInstanceBackupsExpected(veleroBackup velerov1.Backup) int {
	if val, ok := veleroBackup.GetAnnotations()[InstanceBackupsExpectedAnnotation]; ok {
		num, _ := strconv.Atoi(val)
		if num > 0 {
			return num
		}
	}
	return 1
}

// getInstanceBackupMetadata returns metadata about the instance backup for use in creating an
// instance backup.
func getInstanceBackupMetadata(ctx context.Context, k8sClient kubernetes.Interface, ctrlClient ctrlclient.Client, veleroClient veleroclientv1.VeleroV1Interface, cluster *downstreamtypes.Downstream, isScheduled bool) (instanceBackupMetadata, error) {
	now := time.Now().UTC()
	metadata := instanceBackupMetadata{
		backupName:       fmt.Sprintf("backup-%d", now.UnixNano()),
		backupReqestedAt: now,
		kotsadmNamespace: util.PodNamespace,
		apps:             make(map[string]appInstanceBackupMetadata, 0),
		isScheduled:      isScheduled,
	}

	if cluster.SnapshotTTL != "" {
		snapshotTTL, err := time.ParseDuration(cluster.SnapshotTTL)
		if err != nil {
			return metadata, errors.Wrap(err, "failed to parse snapshot ttl")
		}
		metadata.snapshotTTL = snapshotTTL
	}

	kotsadmVeleroBackendStorageLocation, err := kotssnapshot.FindBackupStoreLocation(ctx, k8sClient, veleroClient, metadata.kotsadmNamespace)
	if err != nil {
		return metadata, errors.Wrap(err, "failed to find backupstoragelocations")
	} else if kotsadmVeleroBackendStorageLocation == nil {
		return metadata, errors.New("no backup store location found")
	}
	metadata.backupStorageLocationNamespace = kotsadmVeleroBackendStorageLocation.Namespace

	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		return metadata, errors.Wrap(err, "failed to list installed apps")
	}

	for _, app := range apps {
		downstreams, err := store.GetStore().ListDownstreamsForApp(app.ID)
		if err != nil {
			return metadata, errors.Wrapf(err, "failed to list downstreams for app %s", app.Slug)
		}

		if len(downstreams) == 0 {
			logger.Errorf("No downstreams found for app %s", app.Slug)
			continue
		}

		parentSequence, err := store.GetStore().GetCurrentParentSequence(app.ID, downstreams[0].ClusterID)
		if err != nil {
			return metadata, errors.Wrapf(err, "failed to get current downstream parent sequence for app %s", app.Slug)
		}
		if parentSequence == -1 {
			// no version is deployed for this app yet
			continue
		}

		archiveDir, err := os.MkdirTemp("", "kotsadm")
		if err != nil {
			return metadata, errors.Wrapf(err, "failed to create temp dir for app %s", app.Slug)
		}
		defer func() {
			_ = os.RemoveAll(archiveDir)
		}()

		err = store.GetStore().GetAppVersionArchive(app.ID, parentSequence, archiveDir)
		if err != nil {
			return metadata, errors.Wrapf(err, "failed to get app version archive for app %s", app.Slug)
		}

		kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
		if err != nil {
			return metadata, errors.Wrap(err, "failed to load kots kinds from path")
		}

		metadata.apps[app.Slug] = appInstanceBackupMetadata{
			app:            app,
			kotsKinds:      kotsKinds,
			parentSequence: parentSequence,
		}

		// optimization as we no longer need the archive dir
		_ = os.RemoveAll(archiveDir)
	}

	metadata.ec, err = getECInstanceBackupMetadata(ctx, ctrlClient)
	if err != nil {
		return metadata, errors.Wrap(err, "failed to get embedded cluster metadata")
	}

	return metadata, nil
}

// getECInstanceBackupMetadata returns metadata about the embedded cluster for use in creating an
// instance backup.
func getECInstanceBackupMetadata(ctx context.Context, ctrlClient ctrlclient.Client) (*ecInstanceBackupMetadata, error) {
	if !util.IsEmbeddedCluster() {
		return nil, nil
	}

	installation, err := embeddedcluster.GetCurrentInstallation(ctx, ctrlClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get current installation: %w", err)
	}

	seaweedFSS3ServiceIP, err := embeddedcluster.GetSeaweedFSS3ServiceIP(ctx, ctrlClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get seaweedfs s3 service ip: %w", err)
	}

	return &ecInstanceBackupMetadata{
		installation:         installation,
		seaweedFSS3ServiceIP: seaweedFSS3ServiceIP,
	}, nil
}

// getInstanceBackupSpec returns the velero backup spec for the instance backup. This is either the
// kotsadm backup or the combined backup if this is not using improved DR.
func getInstanceBackupSpec(ctx context.Context, k8sClient kubernetes.Interface, metadata instanceBackupMetadata, hasAppSpec bool) (*velerov1.Backup, error) {
	// veleroBackup is the kotsadm backup or combined backup if usesImprovedDR is false
	veleroBackup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:         "",
			GenerateName: "instance-",
			Annotations:  map[string]string{},
		},
		Spec: velerov1.BackupSpec{
			StorageLocation:         "default",
			IncludedNamespaces:      []string{},
			ExcludedNamespaces:      []string{},
			IncludeClusterResources: ptr.To(true),
			OrLabelSelectors:        instanceBackupLabelSelectors(metadata.ec != nil),
			OrderedResources:        map[string]string{},
			Hooks: velerov1.BackupHooks{
				Resources: []velerov1.BackupResourceHookSpec{},
			},
		},
	}

	appNamespace := metadata.kotsadmNamespace
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}
	if appNamespace != metadata.kotsadmNamespace {
		veleroBackup.Spec.IncludedNamespaces = append(veleroBackup.Spec.IncludedNamespaces, appNamespace)
	}

	isKurl, err := kurl.IsKurl(k8sClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if cluster is kurl")
	}
	if isKurl {
		veleroBackup.Spec.IncludedNamespaces = append(veleroBackup.Spec.IncludedNamespaces, "kurl")
	}

	isKotsadmClusterScoped := k8sutil.IsKotsadmClusterScoped(ctx, k8sClient, metadata.kotsadmNamespace)
	if !isKotsadmClusterScoped {
		// in minimal rbac, a kotsadm role and rolebinding will exist in the velero namespace to give kotsadm access to velero.
		// we backup and restore those so that restoring to a new cluster won't require that the user provide those permissions again.
		veleroBackup.Spec.IncludedNamespaces = append(veleroBackup.Spec.IncludedNamespaces, metadata.backupStorageLocationNamespace)
	}

	for _, appMeta := range metadata.apps {
		// Don't merge the backup spec if we are using the new improved DR.
		if !hasAppSpec {
			err := mergeAppBackupSpec(veleroBackup, appMeta, metadata.kotsadmNamespace, metadata.ec != nil)
			if err != nil {
				return nil, errors.Wrap(err, "failed to merge app backup spec")
			}
		}
	}

	veleroBackup.Annotations, err = appendCommonAnnotations(ctx, k8sClient, veleroBackup.Annotations, metadata, hasAppSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add annotations to backup")
	}
	if hasAppSpec {
		veleroBackup.Annotations[InstanceBackupTypeAnnotation] = InstanceBackupTypeKotsadm
	} else {
		veleroBackup.Annotations[InstanceBackupTypeAnnotation] = InstanceBackupTypeCombined
	}

	if metadata.ec != nil {
		veleroBackup.Spec.IncludedNamespaces = append(veleroBackup.Spec.IncludedNamespaces, ecIncludedNamespaces(metadata.ec.installation)...)
	}

	if metadata.snapshotTTL > 0 {
		veleroBackup.Spec.TTL = metav1.Duration{
			Duration: metadata.snapshotTTL,
		}
	}

	veleroBackup.Spec.IncludedNamespaces = prepareIncludedNamespaces(veleroBackup.Spec.IncludedNamespaces)

	return veleroBackup, nil
}

// getAppInstanceBackup returns a backup spec only if this is Embedded Cluster and the vendor has
// defined both a backup and restore custom resource (improved DR).
func getAppInstanceBackupSpec(ctx context.Context, k8sClient kubernetes.Interface, metadata instanceBackupMetadata) (*velerov1.Backup, error) {
	if metadata.ec == nil {
		return nil, nil
	}

	var appVeleroBackup *velerov1.Backup

	for slug, appMeta := range metadata.apps {
		// if there is both a backup and a restore spec this is using the new improved DR
		if appMeta.kotsKinds.Backup == nil || appMeta.kotsKinds.Restore == nil {
			continue
		}

		if len(metadata.apps) > 1 {
			return nil, errors.New("cannot create backup for Embedded Cluster with multiple apps")
		}

		backupSpec, err := appMeta.kotsKinds.Marshal("velero.io", "v1", "Backup")
		if err != nil {
			return nil, errors.Wrap(err, "failed to get backup spec from kotskinds")
		}

		if backupSpec == "" {
			return nil, errors.New("backup spec is empty, this is unexpected")
		}

		renderedBackup, err := helper.RenderAppFile(appMeta.app, nil, []byte(backupSpec), appMeta.kotsKinds, metadata.kotsadmNamespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to render backup")
		}
		appVeleroBackup, err = kotsutil.LoadBackupFromContents(renderedBackup)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load backup from contents")
		}

		appVeleroBackup.Name = ""
		appVeleroBackup.GenerateName = slug + "-"

		break
	}

	if appVeleroBackup == nil {
		return nil, nil
	}

	appSequences := map[string]int64{}
	appVersions := map[string]string{}

	for slug, appMeta := range metadata.apps {
		appSequences[slug] = appMeta.parentSequence
		appVersions[slug] = appMeta.kotsKinds.Installation.Spec.VersionLabel
	}

	var err error
	appVeleroBackup.Annotations, err = appendCommonAnnotations(ctx, k8sClient, appVeleroBackup.Annotations, metadata, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add annotations to application backup")
	}
	appVeleroBackup.Annotations[InstanceBackupTypeAnnotation] = InstanceBackupTypeApp

	appVeleroBackup.Spec.StorageLocation = "default"

	if metadata.snapshotTTL > 0 {
		appVeleroBackup.Spec.TTL = metav1.Duration{
			Duration: metadata.snapshotTTL,
		}
	}

	return appVeleroBackup, nil
}

// mergeAppBackupSpec merges the app backup spec into the velero backup spec when improved DR is
// disabled. Unsupported fields that are intentionally left out because they might break full
// snapshots:
// - includedResources
// - excludedResources
// - labelSelector
func mergeAppBackupSpec(backup *velerov1.Backup, appMeta appInstanceBackupMetadata, kotsadmNamespace string, isEC bool) error {
	backupSpec, err := appMeta.kotsKinds.Marshal("velero.io", "v1", "Backup")
	if err != nil {
		return errors.Wrap(err, "failed to get backup spec from kotskinds")
	}

	if backupSpec == "" {
		// If this is Embedded Cluster, backups are always enabled and we must include the
		// namespace.
		if isEC {
			backup.Spec.IncludedNamespaces = append(backup.Spec.IncludedNamespaces, appMeta.kotsKinds.KotsApplication.Spec.AdditionalNamespaces...)
		}
		return nil
	}

	renderedBackup, err := helper.RenderAppFile(appMeta.app, nil, []byte(backupSpec), appMeta.kotsKinds, kotsadmNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to render backup")
	}
	kotskindsBackup, err := kotsutil.LoadBackupFromContents(renderedBackup)
	if err != nil {
		return errors.Wrap(err, "failed to load backup from contents")
	}

	// included namespaces
	backup.Spec.IncludedNamespaces = append(backup.Spec.IncludedNamespaces, appMeta.kotsKinds.KotsApplication.Spec.AdditionalNamespaces...)
	backup.Spec.IncludedNamespaces = append(backup.Spec.IncludedNamespaces, kotskindsBackup.Spec.IncludedNamespaces...)

	// excluded namespaces
	backup.Spec.ExcludedNamespaces = append(backup.Spec.ExcludedNamespaces, kotskindsBackup.Spec.ExcludedNamespaces...)

	// annotations
	if len(kotskindsBackup.ObjectMeta.Annotations) > 0 {
		if backup.Annotations == nil {
			backup.Annotations = map[string]string{}
		}
		for k, v := range kotskindsBackup.ObjectMeta.Annotations {
			backup.Annotations[k] = v
		}
	}

	// ordered resources
	if len(kotskindsBackup.Spec.OrderedResources) > 0 {
		if backup.Spec.OrderedResources == nil {
			backup.Spec.OrderedResources = map[string]string{}
		}
		for k, v := range kotskindsBackup.Spec.OrderedResources {
			backup.Spec.OrderedResources[k] = v
		}
	}

	// backup hooks
	backup.Spec.Hooks.Resources = append(backup.Spec.Hooks.Resources, kotskindsBackup.Spec.Hooks.Resources...)

	return nil
}

// appendCommonAnnotations appends common annotations to the backup annotations
func appendCommonAnnotations(ctx context.Context, k8sClient kubernetes.Interface, annotations map[string]string, metadata instanceBackupMetadata, hasAppSpec bool) (map[string]string, error) {
	kotsadmImage, err := k8sutil.FindKotsadmImage(k8sClient, metadata.kotsadmNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find kotsadm image")
	}

	snapshotTrigger := "manual"
	if metadata.isScheduled {
		snapshotTrigger = "schedule"
	}

	appSequences := map[string]int64{}
	appVersions := map[string]string{}

	for slug, appMeta := range metadata.apps {
		appSequences[slug] = appMeta.parentSequence
		appVersions[slug] = appMeta.kotsKinds.Installation.Spec.VersionLabel
	}

	// marshal apps sequences map
	b, err := json.Marshal(appSequences)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal app sequences")
	}
	marshalledAppSequences := string(b)

	// marshal apps versions map
	b, err = json.Marshal(appVersions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal app versions")
	}
	marshalledAppVersions := string(b)

	numBackups := 1
	if hasAppSpec {
		numBackups = 2
	}

	if annotations == nil {
		annotations = make(map[string]string, 0)
	}
	annotations["kots.io/snapshot-trigger"] = snapshotTrigger
	annotations["kots.io/snapshot-requested"] = metadata.backupReqestedAt.Format(time.RFC3339)
	annotations["kots.io/instance"] = "true"
	annotations["kots.io/kotsadm-image"] = kotsadmImage
	annotations["kots.io/kotsadm-deploy-namespace"] = metadata.kotsadmNamespace
	annotations["kots.io/apps-sequences"] = marshalledAppSequences
	annotations["kots.io/apps-versions"] = marshalledAppVersions
	annotations["kots.io/is-airgap"] = strconv.FormatBool(kotsadm.IsAirgap())
	embeddedRegistryHost, _, _ := kotsutil.GetEmbeddedRegistryCreds(k8sClient)
	if embeddedRegistryHost != "" {
		annotations["kots.io/embedded-registry"] = embeddedRegistryHost
	}

	// Add improved disaster recovery annotation labels
	annotations[InstanceBackupNameAnnotation] = metadata.backupName
	annotations[InstanceBackupsExpectedAnnotation] = strconv.Itoa(numBackups)

	if metadata.ec != nil {
		annotations = appendECAnnotations(annotations, *metadata.ec)
	}

	return annotations, nil
}

func ListBackupsForApp(ctx context.Context, kotsadmNamespace string, appID string) ([]*types.Backup, error) {
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
			Name:   veleroBackup.Name, // TODO: GetBackupName(veleroBackup),
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

		// TODO: support for improved DR in UI
		if GetInstanceBackupType(veleroBackup) == InstanceBackupTypeApp {
			continue
		}

		backup := types.Backup{
			Name:         veleroBackup.Name, // TODO: GetBackupName(veleroBackup),
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

	// get the backup
	backup, err := veleroClient.Backups(veleroNamespace).Get(ctx, snapshotName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get backup")
	}

	return backup, nil
}

func DeleteBackup(ctx context.Context, kotsadmNamespace string, snapshotName string) error {
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
	veleroDeleteBackupRequest := &velerov1.DeleteBackupRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapshotName,
			Namespace: veleroNamespace,
		},
		Spec: velerov1.DeleteBackupRequestSpec{
			BackupName: snapshotName,
		},
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
		Name:       backup.Name, // TODO: GetBackupName(*backup),
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
			PodName:        backupVolume.Spec.Pod.Name,
			PodNamespace:   backupVolume.Spec.Pod.Namespace,
			PodVolumeName:  backupVolume.Spec.Volume,
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

	errs, warnings, execs, err := parseLogs(gzipReader, DefaultLogParserBufferSize)
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

// appendECAnnotations appends annotations that should be added to an embedded cluster backup
func appendECAnnotations(annotations map[string]string, ecMeta ecInstanceBackupMetadata) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string, 0)
	}

	if ecMeta.seaweedFSS3ServiceIP != "" {
		annotations["kots.io/embedded-cluster-seaweedfs-s3-ip"] = ecMeta.seaweedFSS3ServiceIP
	}

	annotations["kots.io/embedded-cluster"] = "true"
	annotations["kots.io/embedded-cluster-id"] = util.EmbeddedClusterID()
	annotations["kots.io/embedded-cluster-version"] = util.EmbeddedClusterVersion()
	annotations["kots.io/embedded-cluster-is-ha"] = strconv.FormatBool(ecMeta.installation.Spec.HighAvailability)

	if ecMeta.installation.Spec.Network != nil {
		annotations["kots.io/embedded-cluster-pod-cidr"] = ecMeta.installation.Spec.Network.PodCIDR
		annotations["kots.io/embedded-cluster-service-cidr"] = ecMeta.installation.Spec.Network.ServiceCIDR
	}

	if ecMeta.installation.Spec.RuntimeConfig != nil {
		rcAnnotations := ecRuntimeConfigToBackupAnnotations(ecMeta.installation.Spec.RuntimeConfig)
		for k, v := range rcAnnotations {
			annotations[k] = v
		}
	}

	return annotations
}

func ecRuntimeConfigToBackupAnnotations(runtimeConfig *embeddedclusterv1beta1.RuntimeConfigSpec) map[string]string {
	annotations := map[string]string{}

	if runtimeConfig.AdminConsole.Port > 0 {
		annotations["kots.io/embedded-cluster-admin-console-port"] = strconv.Itoa(runtimeConfig.AdminConsole.Port)
	}
	if runtimeConfig.LocalArtifactMirror.Port > 0 {
		annotations["kots.io/embedded-cluster-local-artifact-mirror-port"] = strconv.Itoa(runtimeConfig.LocalArtifactMirror.Port)
	}
	if runtimeConfig.DataDir != "" {
		annotations["kots.io/embedded-cluster-data-dir"] = runtimeConfig.DataDir
	}

	return annotations
}

// ecIncludedNamespaces returns the namespaces that should be included in an embedded cluster backup
func ecIncludedNamespaces(in *embeddedclusterv1beta1.Installation) []string {
	includedNamespaces := []string{"embedded-cluster", "kube-system", "openebs"}
	if in.Spec.AirGap {
		includedNamespaces = append(includedNamespaces, "registry")
		if in.Spec.HighAvailability {
			includedNamespaces = append(includedNamespaces, "seaweedfs")
		}
	}
	return includedNamespaces
}

// Prepares the list of unique namespaces that will be included in a backup. Empty namespaces are excluded.
// If a wildcard is specified, any specific namespaces will not be included since the backup will include all namespaces.
// Velero does not allow for both a wildcard and specific namespaces and will consider the backup invalid if both are present.
func prepareIncludedNamespaces(namespaces []string) []string {
	uniqueNamespaces := make(map[string]bool)
	for _, n := range namespaces {
		if n == "" {
			continue
		} else if n == "*" {
			return []string{n}
		}
		uniqueNamespaces[n] = true
	}

	includedNamespaces := make([]string, len(uniqueNamespaces))
	i := 0
	for k := range uniqueNamespaces {
		includedNamespaces[i] = k
		i++
	}
	return includedNamespaces
}

// excludeShutdownPodsFromBackup will exclude pods that are in a shutdown state from the backup
func excludeShutdownPodsFromBackup(ctx context.Context, clientset kubernetes.Interface, veleroBackup *velerov1.Backup) (err error) {
	selectorMap := map[string]string{
		"status.phase": string(corev1.PodFailed),
	}

	labelSets := []string{}
	staticSet, err := getLabelSetsForLabelSelector(veleroBackup.Spec.LabelSelector)
	if err != nil {
		return errors.Wrap(err, "failed to get label sets for label selector")
	}
	labelSets = append(labelSets, staticSet...)

	for _, sel := range veleroBackup.Spec.OrLabelSelectors {
		orLabelSet, err := getLabelSetsForLabelSelector(sel)
		if err != nil {
			return errors.Wrap(err, "failed to get label sets for or label selector")
		}
		labelSets = append(labelSets, orLabelSet...)
	}

	for _, namespace := range veleroBackup.Spec.IncludedNamespaces {
		if namespace == "*" {
			namespace = "" // specifying an empty ("") namespace in client-go retrieves resources from all namespaces
		}

		podListOption := metav1.ListOptions{
			FieldSelector: fields.SelectorFromSet(selectorMap).String(),
		}

		if len(labelSets) > 0 {
			for _, labelSet := range labelSets {
				podListOption.LabelSelector = labelSet

				if err := excludeShutdownPodsFromBackupInNamespace(ctx, clientset, namespace, podListOption); err != nil {
					return errors.Wrap(err, "failed to exclude shutdown pods from backup")
				}
			}
		} else {
			if err := excludeShutdownPodsFromBackupInNamespace(ctx, clientset, namespace, podListOption); err != nil {
				return errors.Wrap(err, "failed to exclude shutdown pods from backup")
			}
		}
	}

	return nil
}

// excludeShutdownPodsFromBackupInNamespace will exclude pods that are in a shutdown state from the backup in a specific namespace
func excludeShutdownPodsFromBackupInNamespace(ctx context.Context, clientset kubernetes.Interface, namespace string, failedPodListOptions metav1.ListOptions) error {
	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, failedPodListOptions)
	if err != nil {
		return errors.Wrapf(err, "failed to list pods in namespace %s", namespace)
	}

	if podList == nil {
		return nil
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodFailed && pod.Status.Reason == "Shutdown" {
			logger.Infof("Excluding pod %s in namespace %s from backup", pod.Name, namespace)
			// add velero.io/exclude-from-backup=true label to pod
			if pod.Labels == nil {
				pod.Labels = map[string]string{}
			}

			pod.Labels[kotsadmtypes.ExcludeKey] = kotsadmtypes.ExcludeValue
			_, err := clientset.CoreV1().Pods(pod.Namespace).Update(ctx, &pod, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update pod %s in namespace %s", pod.Name, pod.Namespace)
			}
		}
	}
	return nil
}

func instanceBackupLabelSelectors(isEmbeddedCluster bool) []*metav1.LabelSelector {
	if isEmbeddedCluster { // only DR on embedded-cluster
		return []*metav1.LabelSelector{
			{
				MatchLabels: map[string]string{},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "replicated.com/disaster-recovery",
						Operator: metav1.LabelSelectorOpIn,
						Values: []string{
							"infra",
							"app",
							"ec-install",
						},
					},
				},
			},
			{
				// we cannot add new labels to the docker-registry chart as of May 7th 2024
				// so we need to add a label selector for the docker-registry app
				// https://github.com/twuni/docker-registry.helm/blob/main/templates/deployment.yaml
				MatchLabels: map[string]string{
					"app": "docker-registry",
				},
			},
			{
				// we cannot add new labels to the seaweedfs chart as of June 6th 2024
				// so we need to add a label selector for the seaweedfs app
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "seaweedfs",
				},
			},
		}
	}

	return []*metav1.LabelSelector{
		{
			MatchLabels: map[string]string{
				kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
			},
		},
	}
}

func getLabelSetsForLabelSelector(labelSelector *metav1.LabelSelector) ([]string, error) {
	if labelSelector == nil {
		return nil, nil
	}

	labelSets := []string{}
	if labelSelector.MatchLabels != nil && len(labelSelector.MatchLabels) > 0 {
		labelSets = append(labelSets, labels.SelectorFromSet(labelSelector.MatchLabels).String())
	}
	for _, expr := range labelSelector.MatchExpressions {
		if expr.Operator != metav1.LabelSelectorOpIn {
			return nil, fmt.Errorf("unsupported operator %s in label selector %q", expr.Operator, labelSelector.String())
		}
		for _, value := range expr.Values {
			labelSets = append(labelSets, fmt.Sprintf("%s=%s", expr.Key, value))
		}
	}
	return labelSets, nil
}
