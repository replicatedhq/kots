package socketservice

import (
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
	appstatustypes "github.com/replicatedhq/kots/pkg/api/appstatus/types"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/app"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/appstatus"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	identitytypes "github.com/replicatedhq/kots/pkg/identity/types"
	downstream "github.com/replicatedhq/kots/pkg/kotsadmdownstream"
	snapshot "github.com/replicatedhq/kots/pkg/kotsadmsnapshot"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/socket"
	"github.com/replicatedhq/kots/pkg/socket/transport"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	supportbundletypes "github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/pkg/version"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

type ClusterSocket struct {
	ClusterID             string
	SocketID              string
	SentPreflightURLs     map[string]bool
	LastDeployedSequences map[string]int64
}

type DeployArgs struct {
	AppID                string   `json:"app_id"`
	AppSlug              string   `json:"app_slug"`
	KubectlVersion       string   `json:"kubectl_version"`
	AdditionalNamespaces []string `json:"additional_namespaces"`
	ImagePullSecret      string   `json:"image_pull_secret"`
	Namespace            string   `json:"namespace"`
	PreviousManifests    string   `json:"previous_manifests"`
	Manifests            string   `json:"manifests"`
	Wait                 bool     `json:"wait"`
	ResultCallback       string   `json:"result_callback"`
	ClearNamespaces      []string `json:"clear_namespaces"`
	ClearPVCs            bool     `json:"clear_pvcs"`
	AnnotateSlug         bool     `json:"annotate_slug"`
}

type AppInformersArgs struct {
	AppID     string   `json:"app_id"`
	Informers []string `json:"informers"`
}

type SupportBundleArgs struct {
	URI string `json:"uri"`
}

var server *socket.Server
var clusterSocketHistory = []*ClusterSocket{}
var socketMtx sync.Mutex

// SocketService uses special cluster authorization
func Start() *socket.Server {
	logger.Debug("starting socket service")

	server = socket.NewServer(transport.GetDefaultWebsocketTransport())

	server.On(socket.OnConnection, func(c *socket.Channel, args interface{}) {
		socketMtx.Lock()
		defer socketMtx.Unlock()

		clusterID, err := store.GetStore().GetClusterIDFromDeployToken(c.RequestURL().Query().Get("token"))
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get cluster id from deploy token"))
			return
		}

		logger.Info(fmt.Sprintf("Cluster %s connected to the socket service", clusterID))
		c.Join(clusterID)

		clusterSocket := &ClusterSocket{
			ClusterID:             clusterID,
			SocketID:              c.Id(),
			SentPreflightURLs:     make(map[string]bool, 0),
			LastDeployedSequences: make(map[string]int64, 0),
		}
		clusterSocketHistory = append(clusterSocketHistory, clusterSocket)
	})

	server.On(socket.OnDisconnection, func(c *socket.Channel) {
		socketMtx.Lock()
		defer socketMtx.Unlock()

		updatedClusterSocketHistory := []*ClusterSocket{}
		for _, clusterSocket := range clusterSocketHistory {
			if clusterSocket.SocketID != c.Id() {
				updatedClusterSocketHistory = append(updatedClusterSocketHistory, clusterSocket)
			}
		}
		clusterSocketHistory = updatedClusterSocketHistory
	})

	startLoop(deployLoop, 1)
	startLoop(supportBundleLoop, 1)
	startLoop(restoreLoop, 1)

	return server
}

func startLoop(fn func(), intervalInSeconds time.Duration) {
	go func() {
		for {
			fn()
			time.Sleep(time.Second * intervalInSeconds)
		}
	}()
}

func deployLoop() {
	for _, clusterSocket := range clusterSocketHistory {
		apps, err := store.GetStore().ListAppsForDownstream(clusterSocket.ClusterID)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to list installed apps for downstream"))
			continue
		}

		for _, a := range apps {
			if err := processDeploySocketForApp(clusterSocket, a); err != nil {
				logger.Error(errors.Wrapf(err, "failed to run deploy loop for app %s in cluster %s", a.ID, clusterSocket.ClusterID))
				continue
			}
		}
	}
}

func processDeploySocketForApp(clusterSocket *ClusterSocket, a *apptypes.App) error {
	if a.RestoreInProgressName != "" {
		return nil
	}

	deployedVersion, err := downstream.GetCurrentVersion(a.ID, clusterSocket.ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current downstream version")
	}

	if deployedVersion == nil {
		return nil
	}

	if value, ok := clusterSocket.LastDeployedSequences[a.ID]; ok && value == deployedVersion.ParentSequence {
		// this version is already the currently deployed version
		return nil
	}

	d, err := store.GetStore().GetDownstream(clusterSocket.ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get downstream")
	}

	var deployError error
	defer func() {
		if deployError != nil {
			err := downstream.UpdateDownstreamStatus(a.ID, deployedVersion.Sequence, "failed", deployError.Error())
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

	builder, err := render.NewBuilder(kotsKinds, registrySettings, a.Slug, deployedVersion.Sequence, a.IsAirgap)
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
	previouslyDeployedSequence, err := downstream.GetPreviouslyDeployedSequence(a.ID, clusterSocket.ClusterID)
	if err != nil {
		deployError = errors.Wrap(err, "failed to get previously deployed sequence")
		return deployError
	}
	if previouslyDeployedSequence != -1 {
		previouslyDeployedParentSequence, err := downstream.GetParentSequenceForSequence(a.ID, clusterSocket.ClusterID, previouslyDeployedSequence)
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
		}
	}

	deployArgs := DeployArgs{
		AppID:                a.ID,
		AppSlug:              a.Slug,
		KubectlVersion:       kotsKinds.KotsApplication.Spec.KubectlVersion,
		AdditionalNamespaces: kotsKinds.KotsApplication.Spec.AdditionalNamespaces,
		ImagePullSecret:      imagePullSecret,
		Namespace:            ".",
		Manifests:            base64EncodedManifests,
		PreviousManifests:    base64EncodedPreviousManifests,
		ResultCallback:       "/api/v1/deploy/result",
		Wait:                 false,
		AnnotateSlug:         os.Getenv("ANNOTATE_SLUG") != "",
	}

	c, err := server.GetChannel(clusterSocket.SocketID)
	if err != nil {
		return errors.Wrap(err, "failed to get socket channel from server")
	}
	// Event is sent here
	c.Emit("deploy", deployArgs)

	socketMtx.Lock()
	clusterSocket.LastDeployedSequences[a.ID] = deployedVersion.ParentSequence
	socketMtx.Unlock()

	renderedInformers := []string{}

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
			renderedInformers = append(renderedInformers, renderedInformer)
		}
	}

	if identitydeploy.IsEnabled(kotsKinds.Identity, kotsKinds.IdentityConfig) {
		renderedInformers = append(renderedInformers, fmt.Sprintf("deployment/%s", identitytypes.DeploymentName(a.Slug)))
	}

	if len(renderedInformers) > 0 {
		// send to kots operator
		appInformersArgs := AppInformersArgs{
			AppID:     a.ID,
			Informers: renderedInformers,
		}
		c.Emit("appInformers", appInformersArgs)
	} else {
		// no informers, set state to ready
		defaultReadyState := []appstatustypes.ResourceState{
			{
				Kind:      "EMPTY",
				Name:      "EMPTY",
				Namespace: "EMPTY",
				State:     appstatustypes.StateReady,
			},
		}
		err := appstatus.Set(a.ID, defaultReadyState, time.Now())
		if err != nil {
			return errors.Wrap(err, "failed to set app status")
		}
	}

	return nil
}

func supportBundleLoop() {
	for _, clusterSocket := range clusterSocketHistory {
		apps, err := store.GetStore().ListAppsForDownstream(clusterSocket.ClusterID)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to list apps for cluster"))
		}
		pendingSupportBundles := []*supportbundletypes.PendingSupportBundle{}
		for _, app := range apps {
			appPendingSupportBundles, err := store.GetStore().ListPendingSupportBundlesForApp(app.ID)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to list pending support bundles for app"))
				continue
			}

			pendingSupportBundles = append(pendingSupportBundles, appPendingSupportBundles...)
		}

		for _, sb := range pendingSupportBundles {
			if err := processSupportBundle(clusterSocket, *sb); err != nil {
				logger.Error(errors.Wrapf(err, "failed to process support bundle %s for app %s", sb.ID, sb.AppID))
				continue
			}
		}
	}
}

func processSupportBundle(clusterSocket *ClusterSocket, pendingSupportBundle supportbundletypes.PendingSupportBundle) error {
	a, err := store.GetStore().GetApp(pendingSupportBundle.AppID)
	if err != nil {
		return errors.Wrapf(err, "failed to get app %s", pendingSupportBundle.AppID)
	}

	c, err := server.GetChannel(clusterSocket.SocketID)
	if err != nil {
		return errors.Wrap(err, "failed to get socket channel from server")
	}

	sequence := int64(0)

	currentVersion, err := downstream.GetCurrentVersion(a.ID, clusterSocket.ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current downstream version")
	}
	if currentVersion != nil {
		sequence = currentVersion.Sequence
	}

	archivePath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archivePath)

	err = store.GetStore().GetAppVersionArchive(a.ID, sequence, archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to get current archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to load current kotskinds")
	}

	err = supportbundle.CreateRenderedSpec(a.ID, sequence, "", true, kotsKinds)
	if err != nil {
		return errors.Wrap(err, "failed to create rendered support bundle spec")
	}

	supportBundleArgs := SupportBundleArgs{
		URI: supportbundle.GetSpecURI(a.Slug),
	}
	c.Emit("supportbundle", supportBundleArgs)

	if err := supportbundle.ClearPending(pendingSupportBundle.ID); err != nil {
		return errors.Wrap(err, "failed to clear pending support bundle")
	}

	return nil
}

func restoreLoop() {
	for _, clusterSocket := range clusterSocketHistory {
		apps, err := store.GetStore().ListAppsForDownstream(clusterSocket.ClusterID)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to list installed apps for downstream"))
			continue
		}

		for _, a := range apps {
			if err := processRestoreForApp(clusterSocket, a); err != nil {
				logger.Error(errors.Wrapf(err, "failed to handle restoe for app %s", a.ID))
				continue
			}
		}
	}
}

func processRestoreForApp(clusterSocket *ClusterSocket, a *apptypes.App) error {
	if a.RestoreInProgressName == "" {
		return nil
	}

	switch a.RestoreUndeployStatus {
	case apptypes.UndeployInProcess:
		// no-op
		break

	case apptypes.UndeployCompleted:
		if err := handleUndeployCompleted(clusterSocket, a); err != nil {
			return errors.Wrap(err, "failed to handle undeploy completed")
		}
		break

	case apptypes.UndeployFailed:
		// no-op
		break

	default:
		d, err := store.GetStore().GetDownstream(clusterSocket.ClusterID)
		if err != nil {
			return errors.Wrap(err, "failed to get downstream")
		}

		if err := undeployApp(a, d, clusterSocket); err != nil {
			return errors.Wrap(err, "failed to undeploy app")
		}
		break
	}

	return nil
}

func handleUndeployCompleted(clusterSocket *ClusterSocket, a *apptypes.App) error {
	snapshotName := a.RestoreInProgressName
	restoreName := a.RestoreInProgressName

	backup, err := snapshot.GetBackup(snapshotName)
	if err != nil {
		return errors.Wrap(err, "failed to get backup")
	}
	if backup.Annotations["kots.io/instance"] == "true" {
		restoreName = fmt.Sprintf("%s.%s", snapshotName, a.Slug)
	}

	restore, err := snapshot.GetRestore(restoreName)
	if err != nil {
		return errors.Wrap(err, "failed to get restore")
	}

	if restore == nil {
		return errors.Wrap(startVeleroRestore(snapshotName, a.Slug), "failed to start velero restore")
	}

	return errors.Wrap(checkRestoreComplete(clusterSocket, a, restore), "failed to check restore complete")
}

func startVeleroRestore(snapshotName string, appSlug string) error {
	logger.Info(fmt.Sprintf("creating velero restore object from snapshot %s", snapshotName))

	if err := snapshot.CreateApplicationRestore(snapshotName, appSlug); err != nil {
		return errors.Wrap(err, "failed to create restore")
	}

	return nil
}

func checkRestoreComplete(clusterSocket *ClusterSocket, a *apptypes.App, restore *velerov1.Restore) error {
	switch restore.Status.Phase {
	case velerov1.RestorePhaseCompleted:
		backup, err := snapshot.GetBackup(restore.Spec.BackupName)
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

		logger.Info(fmt.Sprintf("restore complete, re-deploying version %d", sequence))

		if err := RedeployAppVersion(a.ID, sequence, clusterSocket); err != nil {
			return errors.Wrap(err, "failed to redeploy app version")
		}

		if err := createSupportBundle(a.ID, sequence, "", true); err != nil {
			// support bundle is not essential.  keep processing restore status
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

func createSupportBundle(appID string, sequence int64, origin string, inCluster bool) error {
	archivePath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archivePath)

	err = store.GetStore().GetAppVersionArchive(appID, sequence, archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to get current archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to load current kotskinds")
	}

	err = supportbundle.CreateRenderedSpec(appID, sequence, origin, inCluster, kotsKinds)
	if err != nil {
		return errors.Wrap(err, "failed to create rendered support bundle spec")
	}

	return nil
}

func undeployApp(a *apptypes.App, d *downstreamtypes.Downstream, clusterSocket *ClusterSocket) error {
	deployedVersion, err := downstream.GetCurrentVersion(a.ID, d.ClusterID)
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

	backup, err := snapshot.GetBackup(a.RestoreInProgressName)
	if err != nil {
		return errors.Wrap(err, "failed to get backup")
	}

	args := DeployArgs{
		AppID:             a.ID,
		AppSlug:           a.Slug,
		KubectlVersion:    kotsKinds.KotsApplication.Spec.KubectlVersion,
		Namespace:         ".",
		Manifests:         "",
		PreviousManifests: base64EncodedManifests,
		ResultCallback:    "/api/v1/undeploy/result",
		Wait:              true,
		ClearNamespaces:   backup.Spec.IncludedNamespaces,
		ClearPVCs:         true,
	}

	c, err := server.GetChannel(clusterSocket.SocketID)
	if err != nil {
		return errors.Wrap(err, "failed to get socket channel from server")
	}
	c.Emit("deploy", args)

	if err := app.SetRestoreUndeployStatus(a.ID, apptypes.UndeployInProcess); err != nil {
		return errors.Wrap(err, "failed to set restore undeploy status")
	}

	return nil
}

// RedeployAppVersion will force trigger a redeploy of the app version, even if it's currently deployed
// if clusterSocket is nil, a redeploy to all the cluster sockets (downstreams - which theoratically should always be 1) will be triggered
func RedeployAppVersion(appID string, sequence int64, clusterSocket *ClusterSocket) error {
	if err := version.DeployVersion(appID, sequence); err != nil {
		return errors.Wrap(err, "failed to deploy version")
	}

	socketMtx.Lock()
	defer socketMtx.Unlock()

	if clusterSocket != nil {
		delete(clusterSocket.LastDeployedSequences, appID)
	} else {
		for _, clusterSocket := range clusterSocketHistory {
			delete(clusterSocket.LastDeployedSequences, appID)
		}
	}

	return nil
}
