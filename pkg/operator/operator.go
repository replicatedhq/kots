package operator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/app"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/apparchive"
	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	identitytypes "github.com/replicatedhq/kots/pkg/identity/types"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	snapshot "github.com/replicatedhq/kots/pkg/kotsadmsnapshot"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/operator/client"
	operatortypes "github.com/replicatedhq/kots/pkg/operator/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/render"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	supportbundletypes "github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kotskinds/multitype"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	operator *Operator
)

type Operator struct {
	client       client.ClientInterface
	store        store.Store
	clusterToken string
	clusterID    string
	deployMtxs   map[string]*sync.Mutex // key is app id
	k8sClientset kubernetes.Interface
}

func Init(client client.ClientInterface, store store.Store, clusterToken string, k8sClientset kubernetes.Interface) *Operator {
	operator = &Operator{
		client:       client,
		store:        store,
		clusterToken: clusterToken,
		deployMtxs:   map[string]*sync.Mutex{},
		k8sClientset: k8sClientset,
	}
	return operator
}

func MustGetOperator() *Operator {
	if operator != nil {
		return operator
	}
	panic("operator not initialized")
}

func (o *Operator) Start() error {
	logger.Debug("starting the operator")

	if err := o.client.Init(); err != nil {
		return errors.Wrap(err, "failed to initialize the operator client")
	}

	id, err := o.store.GetClusterIDFromDeployToken(o.clusterToken)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster id from deploy token")
	}
	o.clusterID = id

	go o.resumeInformers()
	go o.resumeDeployments()
	startLoop(o.restoreLoop, 2)

	return nil
}

func (o *Operator) Shutdown() {
	if o.client == nil {
		return
	}
	o.client.Shutdown()
}

func startLoop(fn func(), intervalInSeconds time.Duration) {
	go func() {
		for {
			fn()
			time.Sleep(time.Second * intervalInSeconds)
		}
	}()
}

func (o *Operator) resumeDeployments() {
	apps, err := o.store.ListAppsForDownstream(o.clusterID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list installed apps for downstream"))
		return
	}

	for _, a := range apps {
		if _, err := o.resumeDeployment(a); err != nil {
			logger.Error(errors.Wrapf(err, "failed to resume deployment for app %s in cluster %s", a.ID, o.clusterID))
		}
	}
}

func (o *Operator) resumeDeployment(a *apptypes.App) (bool, error) {
	if a.RestoreInProgressName != "" {
		return false, nil
	}

	deployedVersion, err := o.store.GetCurrentDownstreamVersion(a.ID, o.clusterID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get current downstream version")
	} else if deployedVersion == nil {
		return false, nil
	}

	switch deployedVersion.Status {
	case storetypes.VersionDeployed, storetypes.VersionFailed:
		// deploying this version was already attempted
		return false, nil
	}

	if _, err := o.DeployApp(a.ID, deployedVersion.ParentSequence); err != nil {
		return false, errors.Wrap(err, "failed to deploy version")
	}

	return true, nil
}

func (o *Operator) DeployApp(appID string, sequence int64) (deployed bool, deployError error) {
	if _, ok := o.deployMtxs[appID]; !ok {
		o.deployMtxs[appID] = &sync.Mutex{}
	}
	o.deployMtxs[appID].Lock()
	defer o.deployMtxs[appID].Unlock()

	if err := o.store.SetDownstreamVersionStatus(appID, sequence, storetypes.VersionDeploying, ""); err != nil {
		return false, errors.Wrap(err, "failed to update downstream status")
	}

	if os.Getenv("KOTSADM_ENV") != "test" {
		go func() {
			err := reporting.GetReporter().SubmitAppInfo(appID)
			if err != nil {
				logger.Debugf("failed to submit initial app info: %v", err)
			}
		}()

		defer func() {
			err := reporting.GetReporter().SubmitAppInfo(appID)
			if err != nil {
				logger.Debugf("failed to submit final app info: %v", err)
			}
		}()
	}

	defer func() {
		if deployError != nil {
			err := o.store.SetDownstreamVersionStatus(appID, sequence, storetypes.VersionFailed, deployError.Error())
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to update downstream status"))
			}
			return
		}
		if !deployed {
			err := o.store.SetDownstreamVersionStatus(appID, sequence, storetypes.VersionFailed, "")
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to update downstream status"))
			}
			return
		}
		err := o.store.SetDownstreamVersionStatus(appID, sequence, storetypes.VersionDeployed, "")
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to update downstream status"))
		}
	}()

	app, err := o.store.GetApp(appID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get app")
	}

	if app.RestoreInProgressName != "" {
		return false, errors.Errorf("failed to deploy version %d because app restore is already in progress", sequence)
	}

	downstreams, err := o.store.GetDownstream(o.clusterID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get downstream")
	}

	deployedVersionArchive, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return false, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(deployedVersionArchive)

	err = o.store.GetAppVersionArchive(app.ID, sequence, deployedVersionArchive)
	if err != nil {
		return false, errors.Wrap(err, "failed to get app version archive")
	}

	// ensure disaster recovery label transformer in midstream
	additionalLabels := map[string]string{
		"kots.io/app-slug": app.Slug,
	}
	if err := midstream.EnsureDisasterRecoveryLabelTransformer(deployedVersionArchive, additionalLabels); err != nil {
		return false, errors.Wrap(err, "failed to ensure disaster recovery label transformer")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(filepath.Join(deployedVersionArchive, "upstream"))
	if err != nil {
		return false, errors.Wrap(err, "failed to load kotskinds")
	}

	registrySettings, err := o.store.GetRegistryDetailsForApp(app.ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get registry settings for app")
	}

	if err := o.ensureKotsadmApplicationMetadataConfigMap(app, sequence, util.PodNamespace, kotsKinds, registrySettings); err != nil {
		return false, errors.Wrap(err, "failed to ensure kotsadm application metadata configmap")
	}

	builder, err := render.NewBuilder(kotsKinds, registrySettings, app.Slug, sequence, app.IsAirgap, util.PodNamespace)
	if err != nil {
		return false, errors.Wrap(err, "failed to get template builder")
	}

	if kotsKinds.V1Beta1HelmCharts != nil {
		for i, helmChart := range kotsKinds.V1Beta1HelmCharts.Items {
			renderedNamespace, err := builder.String(helmChart.Spec.Namespace)
			if err != nil {
				return false, errors.Wrapf(err, "failed to render namespace %s for chart %s", helmChart.Spec.Namespace, helmChart.GetReleaseName())
			}
			kotsKinds.V1Beta1HelmCharts.Items[i].Spec.Namespace = renderedNamespace

			for j, upgradeFlag := range helmChart.Spec.HelmUpgradeFlags {
				renderedUpgradeFlag, err := builder.String(upgradeFlag)
				if err != nil {
					return false, errors.Wrapf(err, "failed to render upgrade flag %s for chart %s", upgradeFlag, helmChart.GetReleaseName())
				}
				kotsKinds.V1Beta1HelmCharts.Items[i].Spec.HelmUpgradeFlags[j] = renderedUpgradeFlag
			}
		}
	}

	if kotsKinds.V1Beta2HelmCharts != nil {
		for i, helmChart := range kotsKinds.V1Beta2HelmCharts.Items {
			renderedNamespace, err := builder.String(helmChart.Spec.Namespace)
			if err != nil {
				return false, errors.Wrapf(err, "failed to render namespace %s for chart %s", helmChart.Spec.Namespace, helmChart.GetReleaseName())
			}
			kotsKinds.V1Beta2HelmCharts.Items[i].Spec.Namespace = renderedNamespace

			for j, upgradeFlag := range helmChart.Spec.HelmUpgradeFlags {
				renderedUpgradeFlag, err := builder.String(upgradeFlag)
				if err != nil {
					return false, errors.Wrapf(err, "failed to render upgrade flag %s for chart %s", upgradeFlag, helmChart.GetReleaseName())
				}
				kotsKinds.V1Beta2HelmCharts.Items[i].Spec.HelmUpgradeFlags[j] = renderedUpgradeFlag
			}
		}
	}

	requireIdentityProvider := false
	if kotsKinds.Identity != nil {
		if kotsKinds.Identity.Spec.RequireIdentityProvider.Type == multitype.String {
			requireIdentityProvider, err = builder.Bool(kotsKinds.Identity.Spec.RequireIdentityProvider.StrVal, false)
			if err != nil {
				return false, errors.Wrap(err, "failed to build kotsv1beta1.Identity.spec.requireIdentityProvider")
			}
		} else {
			requireIdentityProvider = kotsKinds.Identity.Spec.RequireIdentityProvider.BoolVal
		}
	}

	if requireIdentityProvider && !identitydeploy.IsEnabled(kotsKinds.Identity, kotsKinds.IdentityConfig) {
		return false, errors.New("identity service is required but is not enabled")
	}

	kustomizeBinPath := kotsKinds.GetKustomizeBinaryPath()

	renderedManifests, _, err := apparchive.GetRenderedApp(deployedVersionArchive, downstreams.Name, kustomizeBinPath)
	if err != nil {
		return false, errors.Wrap(err, "failed to get rendered app")
	}
	base64EncodedManifests := base64.StdEncoding.EncodeToString(renderedManifests)

	v1beta1ChartsArchive, _, err := apparchive.GetRenderedV1Beta1ChartsArchive(deployedVersionArchive, downstreams.Name, kustomizeBinPath)
	if err != nil {
		return false, errors.Wrap(err, "failed to get rendered charts archive")
	}

	v1beta2ChartsArchive, err := apparchive.GetV1Beta2ChartsArchive(deployedVersionArchive)
	if err != nil {
		return false, errors.Wrap(err, "failed to get v1beta2 charts archive")
	}

	imagePullSecrets, err := getImagePullSecrets(deployedVersionArchive)
	if err != nil {
		return false, errors.Wrap(err, "failed to get image pull secrets")
	}

	// get previous manifests (if any)
	var previousKotsKinds *kotsutil.KotsKinds
	base64EncodedPreviousManifests := ""
	previousV1beta1ChartsArchive := []byte{}
	previousV1beta2ChartsArchive := []byte{}
	previouslyDeployedSequence, err := o.store.GetPreviouslyDeployedSequence(app.ID, o.clusterID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get previously deployed sequence")
	}
	if previouslyDeployedSequence != -1 {
		previouslyDeployedParentSequence, err := o.store.GetParentSequenceForSequence(app.ID, o.clusterID, previouslyDeployedSequence)
		if err != nil {
			return false, errors.Wrap(err, "failed to get previously deployed parent sequence")
		}

		if previouslyDeployedParentSequence != -1 {
			previouslyDeployedVersionArchive, err := os.MkdirTemp("", "kotsadm")
			if err != nil {
				return false, errors.Wrap(err, "failed to create temp dir")
			}
			defer os.RemoveAll(previouslyDeployedVersionArchive)

			err = o.store.GetAppVersionArchive(app.ID, previouslyDeployedParentSequence, previouslyDeployedVersionArchive)
			if err != nil {
				return false, errors.Wrap(err, "failed to get previously deployed app version archive")
			}

			previousKotsKinds, err = kotsutil.LoadKotsKindsFromPath(filepath.Join(previouslyDeployedVersionArchive, "upstream"))
			if err != nil {
				return false, errors.Wrap(err, "failed to load kotskinds for previously deployed app version")
			}

			previousRenderedManifests, _, err := apparchive.GetRenderedApp(previouslyDeployedVersionArchive, downstreams.Name, kustomizeBinPath)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to get previously deployed rendered app"))
			} else {
				base64EncodedPreviousManifests = base64.StdEncoding.EncodeToString(previousRenderedManifests)
				previousV1beta1ChartsArchive, _, err = apparchive.GetRenderedV1Beta1ChartsArchive(previouslyDeployedVersionArchive, downstreams.Name, kustomizeBinPath)
				if err != nil {
					return false, errors.Wrap(err, "failed to get previously deployed rendered charts archive")
				}

				previousV1beta2ChartsArchive, err = apparchive.GetV1Beta2ChartsArchive(previouslyDeployedVersionArchive)
				if err != nil {
					return false, errors.Wrap(err, "failed to get previously deployed v1beta2 charts archive")
				}
			}
		}
	}

	if err := o.applyStatusInformers(app, sequence, kotsKinds, builder); err != nil {
		return false, errors.Wrap(err, "failed to apply status informers")
	}

	o.client.ApplyNamespacesInformer(kotsKinds.KotsApplication.Spec.AdditionalNamespaces, imagePullSecrets)
	o.client.ApplyHooksInformer(kotsKinds.KotsApplication.Spec.AdditionalNamespaces)

	deployArgs := operatortypes.DeployAppArgs{
		AppID:                        app.ID,
		AppSlug:                      app.Slug,
		ClusterID:                    o.clusterID,
		Sequence:                     sequence,
		KubectlVersion:               kotsKinds.KotsApplication.Spec.KubectlVersion,
		KustomizeVersion:             kotsKinds.KotsApplication.Spec.KustomizeVersion,
		AdditionalNamespaces:         kotsKinds.KotsApplication.Spec.AdditionalNamespaces,
		ImagePullSecrets:             imagePullSecrets,
		Manifests:                    base64EncodedManifests,
		PreviousManifests:            base64EncodedPreviousManifests,
		V1Beta1ChartsArchive:         v1beta1ChartsArchive,
		PreviousV1Beta1ChartsArchive: previousV1beta1ChartsArchive,
		V1Beta2ChartsArchive:         v1beta2ChartsArchive,
		PreviousV1Beta2ChartsArchive: previousV1beta2ChartsArchive,
		Action:                       "deploy",
		Wait:                         false,
		AnnotateSlug:                 os.Getenv("ANNOTATE_SLUG") != "",
		KotsKinds:                    kotsKinds,
		PreviousKotsKinds:            previousKotsKinds,
	}
	deployed, err = o.client.DeployApp(deployArgs)
	if err != nil {
		return false, errors.Wrap(err, "failed to deploy app")
	}

	return deployed, nil
}

func (o *Operator) applyStatusInformers(a *apptypes.App, sequence int64, kotsKinds *kotsutil.KotsKinds, builder *template.Builder) error {
	renderedInformers := []appstatetypes.StatusInformerString{}

	// deploy status informers
	if len(kotsKinds.KotsApplication.Spec.StatusInformers) > 0 {
		// render status informers
		for _, informer := range kotsKinds.KotsApplication.Spec.StatusInformers {
			renderedInformer, err := builder.String(informer)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to render status informer"))
				continue
			}
			if renderedInformer == "" {
				continue
			}
			renderedInformers = append(renderedInformers, appstatetypes.StatusInformerString(renderedInformer))
		}
	}

	if identitydeploy.IsEnabled(kotsKinds.Identity, kotsKinds.IdentityConfig) {
		renderedInformers = append(renderedInformers, appstatetypes.StatusInformerString(fmt.Sprintf("deployment/%s", identitytypes.DeploymentName(a.Slug))))
	}

	if len(renderedInformers) > 0 {
		informersArgs := operatortypes.AppInformersArgs{
			AppID:     a.ID,
			Informers: renderedInformers,
			Sequence:  sequence,
		}
		o.client.ApplyAppInformers(informersArgs)
	} else {
		// no informers, set state to ready
		defaultReadyState := appstatetypes.ResourceStates{
			{
				Kind:      "EMPTY",
				Name:      "EMPTY",
				Namespace: "EMPTY",
				State:     appstatetypes.StateReady,
			},
		}

		err := o.store.SetAppStatus(a.ID, defaultReadyState, time.Now(), sequence)
		if err != nil {
			return errors.Wrap(err, "failed to set app status")
		}

		go func() {
			err := reporting.GetReporter().SubmitAppInfo(a.ID)
			if err != nil {
				logger.Debugf("failed to submit app info: %v", err)
			}
		}()
	}

	return nil
}

func (o *Operator) resumeInformers() {
	apps, err := o.store.ListAppsForDownstream(o.clusterID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list installed apps for downstream"))
		return
	}
	for _, app := range apps {
		if err := o.resumeInformersForApp(app); err != nil {
			logger.Error(errors.Wrapf(err, "failed to resume status informers for app %s in cluster %s", app.ID, o.clusterID))
		}
	}
}

func (o *Operator) resumeInformersForApp(app *apptypes.App) error {
	deployedVersion, err := o.store.GetCurrentDownstreamVersion(app.ID, o.clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current downstream version")
	} else if deployedVersion == nil {
		return nil
	}
	sequence := deployedVersion.ParentSequence

	logger.Debugf("starting status informers for app %s", app.ID)

	deployedVersionArchive, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(deployedVersionArchive)

	err = o.store.GetAppVersionArchive(app.ID, sequence, deployedVersionArchive)
	if err != nil {
		return errors.Wrap(err, "failed to get app version archive")
	}

	imagePullSecrets, err := getImagePullSecrets(deployedVersionArchive)
	if err != nil {
		return errors.Wrap(err, "failed to get image pull secrets")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(filepath.Join(deployedVersionArchive, "upstream"))
	if err != nil {
		return errors.Wrap(err, "failed to load kotskinds")
	}

	registrySettings, err := o.store.GetRegistryDetailsForApp(app.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get registry settings for app")
	}

	builder, err := render.NewBuilder(kotsKinds, registrySettings, app.Slug, sequence, app.IsAirgap, util.PodNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to get template builder")
	}

	if err := o.applyStatusInformers(app, sequence, kotsKinds, builder); err != nil {
		return errors.Wrapf(err, "failed to apply status informers for app %s", app.ID)
	}

	o.client.ApplyNamespacesInformer(kotsKinds.KotsApplication.Spec.AdditionalNamespaces, imagePullSecrets)
	o.client.ApplyHooksInformer(kotsKinds.KotsApplication.Spec.AdditionalNamespaces)

	return nil
}

func (o *Operator) restoreLoop() {
	apps, err := o.store.ListAppsForDownstream(o.clusterID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list installed apps for downstream"))
		return
	}

	for _, a := range apps {
		if err := o.processRestoreForApp(a); err != nil {
			logger.Error(errors.Wrapf(err, "failed to handle restore for app %s", a.ID))
			continue
		}
	}
}

func (o *Operator) processRestoreForApp(a *apptypes.App) error {
	if a.RestoreInProgressName == "" {
		return nil
	}

	switch a.RestoreUndeployStatus {
	case apptypes.UndeployInProcess:
		// no-op
		break

	case apptypes.UndeployCompleted:
		if err := o.handleUndeployCompleted(a); err != nil {
			return errors.Wrap(err, "failed to handle undeploy completed")
		}
		break

	case apptypes.UndeployFailed:
		// no-op
		break

	default:
		d, err := o.store.GetDownstream(o.clusterID)
		if err != nil {
			return errors.Wrap(err, "failed to get downstream")
		}

		if err := o.UndeployApp(a, d, true); err != nil {
			return errors.Wrap(err, "failed to undeploy app")
		}
		break
	}

	return nil
}

func (o *Operator) handleUndeployCompleted(a *apptypes.App) error {
	snapshotName := a.RestoreInProgressName
	restoreName := a.RestoreInProgressName

	backup, err := snapshot.GetBackup(context.Background(), util.PodNamespace, snapshotName)
	if err != nil {
		return errors.Wrap(err, "failed to get backup")
	}
	if backup.Annotations["kots.io/instance"] == "true" {
		restoreName = fmt.Sprintf("%s.%s", snapshotName, a.Slug)
	}

	restore, err := snapshot.GetRestore(context.Background(), util.PodNamespace, restoreName)
	if err != nil {
		return errors.Wrapf(err, "failed to get restore %q", restoreName)
	}

	if restore == nil {
		return errors.Wrapf(o.startVeleroRestore(snapshotName, a.Slug), "failed to start velero restore %q", restoreName)
	}

	return errors.Wrapf(o.checkRestoreComplete(a, restore), "failed to check restore %q complete", restoreName)
}

func (o *Operator) startVeleroRestore(snapshotName string, appSlug string) error {
	logger.Info(fmt.Sprintf("creating velero restore object from snapshot %s", snapshotName))

	if err := snapshot.CreateApplicationRestore(context.Background(), util.PodNamespace, snapshotName, appSlug); err != nil {
		return errors.Wrap(err, "failed to create restore")
	}

	return nil
}

func (o *Operator) checkRestoreComplete(a *apptypes.App, restore *velerov1.Restore) error {
	switch restore.Status.Phase {
	case velerov1.RestorePhaseCompleted:
		backup, err := snapshot.GetBackup(context.Background(), util.PodNamespace, restore.Spec.BackupName)
		if err != nil {
			return errors.Wrap(err, "failed to get backup")
		}

		backupAnnotations := backup.ObjectMeta.GetAnnotations()
		if backupAnnotations == nil {
			return errors.New("backup is missing required annotations")
		}

		var sequence int64 = 0
		if backupAnnotations["kots.io/instance"] == "true" {
			b, ok := backupAnnotations["kots.io/apps-sequences"]
			if !ok || b == "" {
				return errors.New("instance backup is missing apps sequences annotation")
			}

			var appsSequences map[string]int64
			if err := json.Unmarshal([]byte(b), &appsSequences); err != nil {
				return errors.Wrap(err, "failed to unmarshal apps sequences")
			}

			s, ok := appsSequences[a.Slug]
			if !ok {
				return errors.New("instance backup is missing sequence annotation")
			}
			sequence = s
		} else {
			sequenceStr, ok := backupAnnotations["kots.io/app-sequence"]
			if !ok || sequenceStr == "" {
				return errors.New("backup is missing sequence annotation")
			}

			s, err := strconv.ParseInt(sequenceStr, 10, 64)
			if err != nil {
				return errors.Wrap(err, "failed to parse sequence")
			}
			sequence = s
		}

		logger.Info(fmt.Sprintf("restore complete, marking version %d as deployed", sequence))

		// mark the sequence as deployed so that the admin console does not try to re-deploy it
		if err := o.store.MarkAsCurrentDownstreamVersion(a.ID, sequence); err != nil {
			return errors.Wrap(err, "failed to mark as current downstream version")
		}
		if err := o.store.SetDownstreamVersionStatus(a.ID, sequence, storetypes.VersionDeployed, ""); err != nil {
			logger.Error(errors.Wrap(err, "failed to update downstream status"))
		}

		troubleshootOpts := supportbundletypes.TroubleshootOptions{
			InCluster: true,
		}
		if _, err := supportbundle.CreateSupportBundleDependencies(a, sequence, troubleshootOpts); err != nil {
			// support bundle is not essential. keep processing restore status
			logger.Error(errors.Wrapf(err, "failed to create support bundle for sequence %d post restore", sequence))
		}

		if err := app.ResetRestore(a.ID); err != nil {
			return errors.Wrap(err, "failed to reset restore")
		}
		break

	case velerov1.RestorePhaseFailed, velerov1.RestorePhasePartiallyFailed:
		logger.Info("restore failed, resetting app restore")

		if err := app.ResetRestore(a.ID); err != nil {
			return errors.Wrap(err, "failed to reset restore")
		}
		break

	default:
		// restore is in progress
		break
	}

	return nil
}

func (o *Operator) UndeployApp(a *apptypes.App, d *downstreamtypes.Downstream, isRestore bool) error {
	if _, ok := o.deployMtxs[a.ID]; !ok {
		o.deployMtxs[a.ID] = &sync.Mutex{}
	}
	o.deployMtxs[a.ID].Lock()
	defer o.deployMtxs[a.ID].Unlock()

	deployedVersion, err := o.store.GetCurrentDownstreamVersion(a.ID, d.ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current downstream version")
	}
	if deployedVersion == nil {
		return nil
	}

	deployedVersionArchive, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(deployedVersionArchive)

	err = o.store.GetAppVersionArchive(a.ID, deployedVersion.ParentSequence, deployedVersionArchive)
	if err != nil {
		return errors.Wrap(err, "failed to get app version archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(filepath.Join(deployedVersionArchive, "upstream"))
	if err != nil {
		return errors.Wrap(err, "failed to load kotskinds")
	}

	renderedManifests, _, err := apparchive.GetRenderedApp(deployedVersionArchive, d.Name, kotsKinds.GetKustomizeBinaryPath())
	if err != nil {
		return errors.Wrap(err, "failed to get rendered app")
	}
	base64EncodedManifests := base64.StdEncoding.EncodeToString(renderedManifests)

	v1Beta1ChartsArchive, _, err := apparchive.GetRenderedV1Beta1ChartsArchive(deployedVersionArchive, d.Name, kotsKinds.GetKustomizeBinaryPath())
	if err != nil {
		return errors.Wrap(err, "failed to get v1beta1 charts archive")
	}

	v1Beta2ChartsArchive, err := apparchive.GetV1Beta2ChartsArchive(deployedVersionArchive)
	if err != nil {
		return errors.Wrap(err, "failed to get v1beta2 charts archive")
	}

	var clearNamespaces []string
	var restoreLabelSelector *metav1.LabelSelector

	if isRestore {
		backup, err := snapshot.GetBackup(context.Background(), util.PodNamespace, a.RestoreInProgressName)
		if err != nil {
			return errors.Wrap(err, "failed to get backup")
		}
		clearNamespaces = backup.Spec.IncludedNamespaces

		// merge the backup label selector and the restore label selector so that we only undeploy manifests that are:
		// 1- included in the backup AND
		// 2- are going to be restored
		// a valid use case here is when restoring just an app from a full snapshot because the backup won't have this label in that case.
		// this will be a no-op when restoring from an app (partial) snapshot since the backup will already have this label.
		restoreLabelSelector = backup.Spec.LabelSelector.DeepCopy()
		if restoreLabelSelector == nil {
			restoreLabelSelector = &metav1.LabelSelector{
				MatchLabels: map[string]string{},
			}
		}
		restoreLabelSelector.MatchLabels["kots.io/app-slug"] = a.Slug
	} else {
		clearNamespaces = append(clearNamespaces, util.AppNamespace())
		clearNamespaces = append(clearNamespaces, kotsKinds.KotsApplication.Spec.AdditionalNamespaces...)
	}

	undeployArgs := operatortypes.UndeployAppArgs{
		AppID:                a.ID,
		AppSlug:              a.Slug,
		ClusterID:            o.clusterID,
		KubectlVersion:       kotsKinds.KotsApplication.Spec.KubectlVersion,
		KustomizeVersion:     kotsKinds.KotsApplication.Spec.KustomizeVersion,
		AdditionalNamespaces: kotsKinds.KotsApplication.Spec.AdditionalNamespaces,
		Manifests:            base64EncodedManifests,
		V1Beta1ChartsArchive: v1Beta1ChartsArchive,
		V1Beta2ChartsArchive: v1Beta2ChartsArchive,
		Wait:                 true,
		ClearNamespaces:      clearNamespaces,
		ClearPVCs:            true,
		IsRestore:            isRestore,
		RestoreLabelSelector: restoreLabelSelector,
		KotsKinds:            kotsKinds,
	}

	if isRestore {
		// during a restore, this happens async and progress/status is polled later.
		go o.client.UndeployApp(undeployArgs)

		if err := app.SetRestoreUndeployStatus(a.ID, apptypes.UndeployInProcess); err != nil {
			return errors.Wrap(err, "failed to set restore undeploy status")
		}
	} else {
		err := o.client.UndeployApp(undeployArgs)
		if err != nil {
			return errors.Wrap(err, "failed to undeploy app")
		}
	}

	return nil
}

func deduplicateSecrets(secretSpecs []string) []string {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	uniqueSecrets := map[string]*corev1.Secret{}
	for _, secretSpec := range secretSpecs {
		obj, gvk, err := decode([]byte(secretSpec), nil, nil)
		if err != nil {
			continue
		}

		if gvk.Group != "" || gvk.Version != "v1" || gvk.Kind != "Secret" {
			continue
		}
		secret := obj.(*corev1.Secret)
		uniqueSecrets[secret.Name] = secret
	}

	secretSpecs = []string{}
	for _, secret := range uniqueSecrets {
		s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
		var b bytes.Buffer
		if err := s.Encode(secret, &b); err != nil {
			logger.Error(errors.Wrapf(err, "failed to serialize secret %s", secret.Name))
			continue
		}
		secretSpecs = append(secretSpecs, b.String())
	}

	return secretSpecs
}

func (o *Operator) ensureKotsadmApplicationMetadataConfigMap(app *apptypes.App, sequence int64, namespace string, kotsKinds *kotsutil.KotsKinds, registrySettings registrytypes.RegistrySettings) error {
	renderedKotsAppSpec, err := o.renderKotsApplicationSpec(app, sequence, namespace, kotsKinds, registrySettings, &render.Renderer{})
	if err != nil {
		return errors.Wrap(err, "failed to render kots application spec")
	}

	existingConfigMap, err := o.k8sClientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "kotsadm-application-metadata", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing metadata config map")
		}

		_, err := o.k8sClientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), kotsadmobjects.ApplicationMetadataConfig(renderedKotsAppSpec, namespace, app.UpstreamURI), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create metadata config map")
		}
		return nil
	}

	if existingConfigMap.Data == nil {
		existingConfigMap.Data = map[string]string{}
	}

	existingConfigMap.Data["application.yaml"] = string(renderedKotsAppSpec)
	_, err = o.k8sClientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.Background(), existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}

func (o *Operator) renderKotsApplicationSpec(app *apptypes.App, sequence int64, namespace string, kotsKinds *kotsutil.KotsKinds, registrySettings registrytypes.RegistrySettings, renderer rendertypes.Renderer) ([]byte, error) {
	marshalledKotsAppSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Application")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal kots app spec")
	}

	renderedKotsAppSpec, err := renderer.RenderFile(kotsKinds, registrySettings, app.Slug, sequence, app.IsAirgap, namespace, []byte(marshalledKotsAppSpec))
	if err != nil {
		return nil, errors.Wrap(err, "failed to render preflights")
	}

	return renderedKotsAppSpec, nil
}
