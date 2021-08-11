package operator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/operator/client"
	operatortypes "github.com/replicatedhq/kots/pkg/operator/types"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	supportbundletypes "github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var operatorClient *client.Client
var clusterID string
var lastDeployedSequences map[string]int64
var socketMtx sync.Mutex

func Start(clusterToken string) error {
	logger.Debug("starting the operator")

	operatorClient = &client.Client{
		TargetNamespace:   util.AppNamespace(),
		ExistingInformers: map[string]bool{},
		HookStopChans:     []chan struct{}{},
	}
	if err := operatorClient.Init(); err != nil {
		return errors.Wrap(err, "failed to initialize the operator client")
	}

	id, err := store.GetStore().GetClusterIDFromDeployToken(clusterToken)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster id from deploy token")
	}
	clusterID = id

	lastDeployedSequences = make(map[string]int64, 0)

	startLoop(deployLoop, 1)
	startLoop(restoreLoop, 1)

	return nil
}

func Shutdown() {
	if operatorClient == nil {
		return
	}
	operatorClient.Shutdown()
}

func startLoop(fn func(), intervalInSeconds time.Duration) {
	go func() {
		for {
			fn()
			time.Sleep(time.Second * intervalInSeconds)
		}
	}()
}

// appDeployLoopErrorBackoff is a global map of loggers for each app that deploy loop uses to keep
// track of last time an error was logged and prevent duplicate logging.
var appDeployLoopErrorBackoff = map[string]*util.ErrorBackoff{}

func deployLoop() {
	apps, err := store.GetStore().ListAppsForDownstream(clusterID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list installed apps for downstream"))
		return
	}

	for _, a := range apps {
		deployed, err := processDeployForApp(a)
		if err != nil {
			_, ok := appDeployLoopErrorBackoff[a.ID]
			if !ok {
				appDeployLoopErrorBackoff[a.ID] = &util.ErrorBackoff{MinPeriod: 1 * time.Second, MaxPeriod: 30 * time.Minute}
			}
			appDeployLoopErrorBackoff[a.ID].OnError(err, func() {
				logger.Error(errors.Wrapf(err, "failed to run deploy loop for app %s in cluster %s", a.ID, clusterID))
			})
		} else if deployed {
			logger.Infof("Deploy success for app %s in cluster %s", a.ID, clusterID)
		}
	}
}

func processDeployForApp(a *apptypes.App) (bool, error) {
	if a.RestoreInProgressName != "" {
		return false, nil
	}

	deployedVersion, err := store.GetStore().GetCurrentVersion(a.ID, clusterID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get current downstream version")
	} else if deployedVersion == nil {
		return false, nil
	}

	if value, ok := lastDeployedSequences[a.ID]; ok && value == deployedVersion.ParentSequence {
		// this version is already the currently deployed version
		return false, nil
	}

	if err := deployVersionForApp(a, deployedVersion); err != nil {
		return false, errors.Wrap(err, "failed to deploy version")
	}

	return true, nil
}

func deployVersionForApp(a *apptypes.App, deployedVersion *downstreamtypes.DownstreamVersion) error {
	d, err := store.GetStore().GetDownstream(clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get downstream")
	}

	var deployError error
	defer func() {
		if deployError != nil {
			err := store.GetStore().UpdateDownstreamVersionStatus(a.ID, deployedVersion.Sequence, "failed", deployError.Error())
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to update downstream status"))
			}
		}
	}()

	deployedVersionArchive, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		deployError = errors.Wrap(err, "failed to create temp dir")
		return deployError
	}
	defer os.RemoveAll(deployedVersionArchive)

	err = store.GetStore().GetAppVersionArchive(a.ID, deployedVersion.ParentSequence, deployedVersionArchive)
	if err != nil {
		deployError = errors.Wrap(err, "failed to get app version archive")
		return deployError
	}

	// ensure disaster recovery label transformer in midstream
	additionalLabels := map[string]string{
		"kots.io/app-slug": a.Slug,
	}
	if err := midstream.EnsureDisasterRecoveryLabelTransformer(deployedVersionArchive, additionalLabels); err != nil {
		deployError = errors.Wrap(err, "failed to ensure disaster recovery label transformer")
		return deployError
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(deployedVersionArchive)
	if err != nil {
		deployError = errors.Wrap(err, "failed to load kotskinds")
		return deployError
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get registry settings for app")
	}

	builder, err := render.NewBuilder(kotsKinds, registrySettings, a.Slug, deployedVersion.Sequence, a.IsAirgap, util.PodNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to get template builder")
	}

	requireIdentityProvider := false
	if kotsKinds.Identity != nil {
		if kotsKinds.Identity.Spec.RequireIdentityProvider.Type == multitype.String {
			requireIdentityProvider, err = builder.Bool(kotsKinds.Identity.Spec.RequireIdentityProvider.StrVal, false)
			if err != nil {
				deployError = errors.Wrap(err, "failed to build kotsv1beta1.Identity.spec.requireIdentityProvider")
				return deployError
			}
		} else {
			requireIdentityProvider = kotsKinds.Identity.Spec.RequireIdentityProvider.BoolVal
		}
	}

	if requireIdentityProvider && !identitydeploy.IsEnabled(kotsKinds.Identity, kotsKinds.IdentityConfig) {
		deployError = errors.New("identity service is required but is not enabled")
		return deployError
	}

	cmd := exec.Command(fmt.Sprintf("kustomize%s", kotsKinds.KustomizeVersion()), "build", filepath.Join(deployedVersionArchive, "overlays", "downstreams", d.Name))
	renderedManifests, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
		}
		deployError = errors.Wrap(err, "failed to run kustomize")
		return deployError
	}
	base64EncodedManifests := base64.StdEncoding.EncodeToString(renderedManifests)

	chartArchive, err := renderChartsArchive(deployedVersionArchive, d.Name, kotsKinds.KustomizeVersion())
	if err != nil {
		deployError = errors.Wrap(err, "failed to run kustomize on currently deployed charts")
		return deployError
	}

	imagePullSecret := ""
	secretFilename := filepath.Join(deployedVersionArchive, "overlays", "midstream", "secret.yaml")
	_, err = os.Stat(secretFilename)
	if err != nil && !os.IsNotExist(err) {
		deployError = errors.Wrap(err, "failed to os stat image pull secret file")
		return deployError
	}
	if err == nil {
		b, err := ioutil.ReadFile(secretFilename)
		if err != nil {
			deployError = errors.Wrap(err, "failed to read image pull secret file")
			return deployError
		}
		imagePullSecret = string(b)
	}

	// get previous manifests (if any)
	base64EncodedPreviousManifests := ""
	previouslyDeployedChartArchive := []byte{}
	previouslyDeployedSequence, err := store.GetStore().GetPreviouslyDeployedSequence(a.ID, clusterID)
	if err != nil {
		deployError = errors.Wrap(err, "failed to get previously deployed sequence")
		return deployError
	}
	if previouslyDeployedSequence != -1 {
		previouslyDeployedParentSequence, err := store.GetStore().GetParentSequenceForSequence(a.ID, clusterID, previouslyDeployedSequence)
		if err != nil {
			deployError = errors.Wrap(err, "failed to get previously deployed parent sequence")
			return deployError
		}

		if previouslyDeployedParentSequence != -1 {
			previouslyDeployedVersionArchive, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				deployError = errors.Wrap(err, "failed to create temp dir")
				return deployError
			}
			defer os.RemoveAll(previouslyDeployedVersionArchive)

			err = store.GetStore().GetAppVersionArchive(a.ID, previouslyDeployedParentSequence, previouslyDeployedVersionArchive)
			if err != nil {
				deployError = errors.Wrap(err, "failed to get previously deployed app version archive")
				return deployError
			}

			previousKotsKinds, err := kotsutil.LoadKotsKindsFromPath(previouslyDeployedVersionArchive)
			if err != nil {
				deployError = errors.Wrap(err, "failed to load kotskinds for previously deployed app version")
				return deployError
			}

			cmd := exec.Command(fmt.Sprintf("kustomize%s", previousKotsKinds.KustomizeVersion()), "build", filepath.Join(previouslyDeployedVersionArchive, "overlays", "downstreams", d.Name))
			previousRenderedManifests, err := cmd.Output()
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					err = fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
				}
				deployError = errors.Wrap(err, "failed to run kustomize for previously deployed app version")
				return deployError
			}

			base64EncodedPreviousManifests = base64.StdEncoding.EncodeToString(previousRenderedManifests)
			// Run kustomization on the charts as well
			previouslyDeployedChartArchive, err = renderChartsArchive(previouslyDeployedVersionArchive, d.Name, kotsKinds.KustomizeVersion())
			if err != nil {
				deployError = errors.Wrap(err, "failed to run kustomize on previously deployed charts")
				return deployError
			}

		}
	}

	deployArgs := operatortypes.DeployAppArgs{
		AppID:                a.ID,
		AppSlug:              a.Slug,
		ClusterID:            clusterID,
		Sequence:             deployedVersion.ParentSequence,
		KubectlVersion:       kotsKinds.KotsApplication.Spec.KubectlVersion,
		AdditionalNamespaces: kotsKinds.KotsApplication.Spec.AdditionalNamespaces,
		ImagePullSecret:      imagePullSecret,
		Namespace:            ".",
		Manifests:            base64EncodedManifests,
		PreviousManifests:    base64EncodedPreviousManifests,
		Charts:               chartArchive,
		PreviousCharts:       previouslyDeployedChartArchive,
		Action:               "deploy",
		Wait:                 false,
		AnnotateSlug:         os.Getenv("ANNOTATE_SLUG") != "",
	}
	operatorClient.DeployApp(deployArgs) // this happens async and results are reported once the process is complete

	socketMtx.Lock()
	lastDeployedSequences[a.ID] = deployedVersion.ParentSequence
	socketMtx.Unlock()

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
			Sequence:  deployedVersion.Sequence,
		}
		operatorClient.ApplyAppInformers(informersArgs)
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

		err := store.GetStore().SetAppStatus(a.ID, defaultReadyState, time.Now(), deployedVersion.Sequence)
		if err != nil {
			return errors.Wrap(err, "failed to set app status")
		}
		go reporting.SendAppInfo(a.ID)
	}

	return nil
}

func restoreLoop() {
	apps, err := store.GetStore().ListAppsForDownstream(clusterID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list installed apps for downstream"))
		return
	}

	for _, a := range apps {
		if err := processRestoreForApp(a); err != nil {
			logger.Error(errors.Wrapf(err, "failed to handle restoe for app %s", a.ID))
			continue
		}
	}
}

func processRestoreForApp(a *apptypes.App) error {
	if a.RestoreInProgressName == "" {
		return nil
	}

	switch a.RestoreUndeployStatus {
	case apptypes.UndeployInProcess:
		// no-op
		break

	case apptypes.UndeployCompleted:
		if err := handleUndeployCompleted(a); err != nil {
			return errors.Wrap(err, "failed to handle undeploy completed")
		}
		break

	case apptypes.UndeployFailed:
		// no-op
		break

	default:
		d, err := store.GetStore().GetDownstream(clusterID)
		if err != nil {
			return errors.Wrap(err, "failed to get downstream")
		}

		if err := undeployApp(a, d, true); err != nil {
			return errors.Wrap(err, "failed to undeploy app")
		}
		break
	}

	return nil
}

func handleUndeployCompleted(a *apptypes.App) error {
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
		return errors.Wrap(err, "failed to get restore")
	}

	if restore == nil {
		return errors.Wrap(startVeleroRestore(snapshotName, a.Slug), "failed to start velero restore")
	}

	return errors.Wrap(checkRestoreComplete(a, restore), "failed to check restore complete")
}

func startVeleroRestore(snapshotName string, appSlug string) error {
	logger.Info(fmt.Sprintf("creating velero restore object from snapshot %s", snapshotName))

	if err := snapshot.CreateApplicationRestore(context.Background(), util.PodNamespace, snapshotName, appSlug); err != nil {
		return errors.Wrap(err, "failed to create restore")
	}

	return nil
}

func checkRestoreComplete(a *apptypes.App, restore *velerov1.Restore) error {
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

		// mark the sequence as deployed both in the db and sequences history
		// so that the admin console does not try to re-deploy it
		if err := version.DeployVersion(a.ID, sequence); err != nil {
			return errors.Wrap(err, "failed to mark app version as deployed")
		}
		socketMtx.Lock()
		lastDeployedSequences[a.ID] = sequence
		socketMtx.Unlock()

		troubleshootOpts := supportbundletypes.TroubleshootOptions{
			InCluster: true,
		}
		if _, err := supportbundle.CreateSupportBundleDependencies(a.ID, sequence, troubleshootOpts); err != nil {
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

func undeployApp(a *apptypes.App, d *downstreamtypes.Downstream, isRestore bool) error {
	deployedVersion, err := store.GetStore().GetCurrentVersion(a.ID, d.ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current downstream version")
	}

	deployedVersionArchive, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(deployedVersionArchive)

	err = store.GetStore().GetAppVersionArchive(a.ID, deployedVersion.ParentSequence, deployedVersionArchive)
	if err != nil {
		return errors.Wrap(err, "failed to get app version archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(deployedVersionArchive)
	if err != nil {
		return errors.Wrap(err, "failed to load kotskinds")
	}

	cmd := exec.Command(fmt.Sprintf("kustomize%s", kotsKinds.KustomizeVersion()), "build", filepath.Join(deployedVersionArchive, "overlays", "downstreams", d.Name))
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
		ClusterID:            clusterID,
		KubectlVersion:       kotsKinds.KotsApplication.Spec.KubectlVersion,
		Namespace:            ".",
		Manifests:            "",
		PreviousManifests:    base64EncodedManifests,
		Action:               "undeploy",
		Wait:                 true,
		ClearNamespaces:      backup.Spec.IncludedNamespaces,
		ClearPVCs:            true,
		IsRestore:            isRestore,
		RestoreLabelSelector: restoreLabelSelector,
	}
	operatorClient.DeployApp(undeployArgs)

	if err := app.SetRestoreUndeployStatus(a.ID, apptypes.UndeployInProcess); err != nil {
		return errors.Wrap(err, "failed to set restore undeploy status")
	}

	return nil
}

// RedeployAppVersion will force trigger a redeploy of the app version, even if it's currently deployed
func RedeployAppVersion(appID string, sequence int64) error {
	if err := version.DeployVersion(appID, sequence); err != nil {
		return errors.Wrap(err, "failed to deploy version")
	}

	socketMtx.Lock()
	delete(lastDeployedSequences, appID)
	socketMtx.Unlock()

	return nil
}
