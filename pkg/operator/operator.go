package operator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/app"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	identitytypes "github.com/replicatedhq/kots/pkg/identity/types"
	snapshot "github.com/replicatedhq/kots/pkg/kotsadmsnapshot"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kustomize"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/operator/client"
	operatortypes "github.com/replicatedhq/kots/pkg/operator/types"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	supportbundletypes "github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/util"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
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
}

func Init(client client.ClientInterface, store store.Store, clusterToken string) *Operator {
	operator = &Operator{
		client:       client,
		store:        store,
		clusterToken: clusterToken,
		deployMtxs:   map[string]*sync.Mutex{},
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

	o.startStatusInformers()
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

	deployedVersionArchive, err := ioutil.TempDir("", "kotsadm")
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

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(deployedVersionArchive)
	if err != nil {
		return false, errors.Wrap(err, "failed to load kotskinds")
	}

	registrySettings, err := o.store.GetRegistryDetailsForApp(app.ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get registry settings for app")
	}

	builder, err := render.NewBuilder(kotsKinds, registrySettings, app.Slug, sequence, app.IsAirgap, util.PodNamespace)
	if err != nil {
		return false, errors.Wrap(err, "failed to get template builder")
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

	cmd := exec.Command(kustomizeBinPath, "build", filepath.Join(deployedVersionArchive, "overlays", "downstreams", downstreams.Name))
	renderedManifests, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
		}
		return false, errors.Wrap(err, "failed to run kustomize")
	}
	base64EncodedManifests := base64.StdEncoding.EncodeToString(renderedManifests)

	chartArchive, _, err := kustomize.RenderChartsArchive(deployedVersionArchive, downstreams.Name, kustomizeBinPath)
	if err != nil {
		return false, errors.Wrap(err, "failed to run kustomize on currently deployed charts")
	}

	imagePullSecrets := []string{}
	secretFilename := filepath.Join(deployedVersionArchive, "overlays", "midstream", "secret.yaml")
	_, err = os.Stat(secretFilename)
	if err != nil && !os.IsNotExist(err) {
		return false, errors.Wrap(err, "failed to os stat image pull secret file")
	}
	if err == nil {
		b, err := ioutil.ReadFile(secretFilename)
		if err != nil {
			return false, errors.Wrap(err, "failed to read image pull secret file")
		}
		imagePullSecrets = strings.Split(string(b), "\n---\n")
	}

	chartPullSecrets, err := getChartsImagePullSecrets(deployedVersionArchive)
	if err != nil {
		deployError = errors.Wrap(err, "failed to read image pull secret files from charts")
		return false, deployError
	}
	imagePullSecrets = append(imagePullSecrets, chartPullSecrets...)
	imagePullSecrets = deduplicateSecrets(imagePullSecrets)

	// get previous manifests (if any)
	var previousKotsKinds *kotsutil.KotsKinds
	base64EncodedPreviousManifests := ""
	previouslyDeployedChartArchive := []byte{}
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
			previouslyDeployedVersionArchive, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				return false, errors.Wrap(err, "failed to create temp dir")
			}
			defer os.RemoveAll(previouslyDeployedVersionArchive)

			err = o.store.GetAppVersionArchive(app.ID, previouslyDeployedParentSequence, previouslyDeployedVersionArchive)
			if err != nil {
				return false, errors.Wrap(err, "failed to get previously deployed app version archive")
			}

			previousKotsKinds, err = kotsutil.LoadKotsKindsFromPath(previouslyDeployedVersionArchive)
			if err != nil {
				return false, errors.Wrap(err, "failed to load kotskinds for previously deployed app version")
			}

			cmd := exec.Command(previousKotsKinds.GetKustomizeBinaryPath(), "build", filepath.Join(previouslyDeployedVersionArchive, "overlays", "downstreams", downstreams.Name))
			previousRenderedManifests, err := cmd.Output()
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					err = fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
				}
				logger.Error(errors.Wrap(err, "failed to run kustomize for previously deployed app version."))
			} else {
				base64EncodedPreviousManifests = base64.StdEncoding.EncodeToString(previousRenderedManifests)
				// Run kustomization on the charts as well
				previouslyDeployedChartArchive, _, err = kustomize.RenderChartsArchive(previouslyDeployedVersionArchive, downstreams.Name, kustomizeBinPath)
				if err != nil {
					return false, errors.Wrap(err, "failed to run kustomize on previously deployed charts")
				}
			}
		}
	}

	deployArgs := operatortypes.DeployAppArgs{
		AppID:                app.ID,
		AppSlug:              app.Slug,
		ClusterID:            o.clusterID,
		Sequence:             sequence,
		KubectlVersion:       kotsKinds.KotsApplication.Spec.KubectlVersion,
		KustomizeVersion:     kotsKinds.KotsApplication.Spec.KustomizeVersion,
		AdditionalNamespaces: kotsKinds.KotsApplication.Spec.AdditionalNamespaces,
		ImagePullSecrets:     imagePullSecrets,
		Namespace:            ".",
		Manifests:            base64EncodedManifests,
		PreviousManifests:    base64EncodedPreviousManifests,
		Charts:               chartArchive,
		PreviousCharts:       previouslyDeployedChartArchive,
		Action:               "deploy",
		Wait:                 false,
		AnnotateSlug:         os.Getenv("ANNOTATE_SLUG") != "",
		KotsKinds:            kotsKinds,
		PreviousKotsKinds:    previousKotsKinds,
	}
	deployed, err = o.client.DeployApp(deployArgs)
	if err != nil {
		return false, errors.Wrap(err, "failed to deploy app")
	}

	if err := o.applyStatusInformers(app, sequence, kotsKinds, builder); err != nil {
		return false, errors.Wrap(err, "failed to apply status informers")
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
		go reporting.SendAppInfo(a.ID)
	}

	return nil
}

func (o *Operator) startStatusInformers() {
	apps, err := o.store.ListAppsForDownstream(o.clusterID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list installed apps for downstream"))
		return
	}
	for _, app := range apps {
		if err := o.startStatusInformersForApp(app); err != nil {
			logger.Error(errors.Wrapf(err, "failed to start status informers for app %s in cluster %s", app.ID, o.clusterID))
		}
	}
}

func (o *Operator) startStatusInformersForApp(app *apptypes.App) error {
	deployedVersion, err := o.store.GetCurrentDownstreamVersion(app.ID, o.clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current downstream version")
	} else if deployedVersion == nil {
		return nil
	}
	sequence := deployedVersion.ParentSequence

	logger.Debugf("starting status informers for app %s", app.ID)

	deployedVersionArchive, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(deployedVersionArchive)

	err = o.store.GetAppVersionArchive(app.ID, sequence, deployedVersionArchive)
	if err != nil {
		return errors.Wrap(err, "failed to get app version archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(deployedVersionArchive)
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
			logger.Error(errors.Wrapf(err, "failed to handle restoe for app %s", a.ID))
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
		downstreams, err := o.store.GetDownstream(o.clusterID)
		if err != nil {
			return errors.Wrap(err, "failed to get downstream")
		}

		if err := o.undeployApp(a, downstreams, true); err != nil {
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

func (o *Operator) undeployApp(a *apptypes.App, d *downstreamtypes.Downstream, isRestore bool) error {
	if _, ok := o.deployMtxs[a.ID]; !ok {
		o.deployMtxs[a.ID] = &sync.Mutex{}
	}
	o.deployMtxs[a.ID].Lock()
	defer o.deployMtxs[a.ID].Unlock()

	deployedVersion, err := o.store.GetCurrentDownstreamVersion(a.ID, d.ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current downstream version")
	}

	deployedVersionArchive, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(deployedVersionArchive)

	err = o.store.GetAppVersionArchive(a.ID, deployedVersion.ParentSequence, deployedVersionArchive)
	if err != nil {
		return errors.Wrap(err, "failed to get app version archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(deployedVersionArchive)
	if err != nil {
		return errors.Wrap(err, "failed to load kotskinds")
	}

	cmd := exec.Command(kotsKinds.GetKustomizeBinaryPath(), "build", filepath.Join(deployedVersionArchive, "overlays", "downstreams", d.Name))
	renderedManifests, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
		}
		return errors.Wrap(err, "failed to run kustomize")
	}
	base64EncodedManifests := base64.StdEncoding.EncodeToString(renderedManifests)

	backup, err := snapshot.GetBackup(context.Background(), util.PodNamespace, a.RestoreInProgressName)
	if err != nil {
		return errors.Wrap(err, "failed to get backup")
	}

	// merge the backup label selector and the restore label selector so that we only undeploy manifests that are:
	// 1- included in the backup AND
	// 2- are going to be restored
	// a valid use case here is when restoring just an app from a full snapshot because the backup won't have this label in that case.
	// this will be a no-op when restoring from an app (partial) snapshot since the backup will already have this label.
	restoreLabelSelector := backup.Spec.LabelSelector.DeepCopy()
	if restoreLabelSelector == nil {
		restoreLabelSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{},
		}
	}
	restoreLabelSelector.MatchLabels["kots.io/app-slug"] = a.Slug

	undeployArgs := operatortypes.DeployAppArgs{
		AppID:                a.ID,
		AppSlug:              a.Slug,
		ClusterID:            o.clusterID,
		KubectlVersion:       kotsKinds.KotsApplication.Spec.KubectlVersion,
		KustomizeVersion:     kotsKinds.KotsApplication.Spec.KustomizeVersion,
		Namespace:            ".",
		Manifests:            "",
		PreviousManifests:    base64EncodedManifests,
		Action:               "undeploy",
		Wait:                 true,
		ClearNamespaces:      backup.Spec.IncludedNamespaces,
		ClearPVCs:            true,
		IsRestore:            isRestore,
		RestoreLabelSelector: restoreLabelSelector,
		KotsKinds:            kotsKinds,
	}
	go o.client.DeployApp(undeployArgs) // this happens async and progress/status is polled later.

	if err := app.SetRestoreUndeployStatus(a.ID, apptypes.UndeployInProcess); err != nil {
		return errors.Wrap(err, "failed to set restore undeploy status")
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
