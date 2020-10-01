package socketservice

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/appstatus"
	appstatustypes "github.com/replicatedhq/kots/kotsadm/pkg/appstatus/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	downstreamtypes "github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/render"
	"github.com/replicatedhq/kots/kotsadm/pkg/snapshot"
	"github.com/replicatedhq/kots/kotsadm/pkg/socket"
	"github.com/replicatedhq/kots/kotsadm/pkg/socket/transport"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle"
	supportbundletypes "github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	"github.com/replicatedhq/kots/pkg/kotsutil"

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

type SocketService struct {
	Server               *socket.Server
	clusterSocketHistory []ClusterSocket
	socketMtx            sync.Mutex
}

// SocketService uses special cluster authorization
func Start() *SocketService {
	logger.Debug("starting socket service")

	service := &SocketService{
		Server:               socket.NewServer(transport.GetDefaultWebsocketTransport()),
		clusterSocketHistory: []ClusterSocket{},
	}

	service.Server.On(socket.OnConnection, func(c *socket.Channel, args interface{}) {
		service.socketMtx.Lock()
		defer service.socketMtx.Unlock()

		clusterID, err := store.GetStore().GetClusterIDFromDeployToken(c.RequestURL().Query().Get("token"))
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get cluster id from deploy token"))
			return
		}

		logger.Info(fmt.Sprintf("Cluster %s connected to the socket service", clusterID))
		c.Join(clusterID)

		clusterSocket := ClusterSocket{
			ClusterID:             clusterID,
			SocketID:              c.Id(),
			SentPreflightURLs:     make(map[string]bool, 0),
			LastDeployedSequences: make(map[string]int64, 0),
		}
		service.clusterSocketHistory = append(service.clusterSocketHistory, clusterSocket)
	})

	service.Server.On(socket.OnDisconnection, func(c *socket.Channel) {
		service.socketMtx.Lock()
		defer service.socketMtx.Unlock()

		updatedClusterSocketHistory := []ClusterSocket{}
		for _, clusterSocket := range service.clusterSocketHistory {
			if clusterSocket.SocketID != c.Id() {
				updatedClusterSocketHistory = append(updatedClusterSocketHistory, clusterSocket)
			}
		}
		service.clusterSocketHistory = updatedClusterSocketHistory
	})

	startLoop(service.deployLoop, 1)
	startLoop(service.supportBundleLoop, 1)
	startLoop(service.restoreLoop, 1)

	return service
}

func startLoop(fn func(), intervalInSeconds time.Duration) {
	go func() {
		for {
			fn()
			time.Sleep(time.Second * intervalInSeconds)
		}
	}()
}

func (s *SocketService) deployLoop() {
	for _, clusterSocket := range s.clusterSocketHistory {
		apps, err := store.GetStore().ListAppsForDownstream(clusterSocket.ClusterID)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to list installed apps for downstream"))
			continue
		}

		for _, a := range apps {
			if err := s.processDeploySocketForApp(clusterSocket, a); err != nil {
				logger.Error(errors.Wrapf(err, "failed to run deploy loop for app %s in cluster %s", a.ID, clusterSocket.ClusterID))
				continue
			}
		}
	}
}

func (s *SocketService) processDeploySocketForApp(clusterSocket ClusterSocket, a *apptypes.App) error {
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

	deployedVersionArchive, err := store.GetStore().GetAppVersionArchive(a.ID, deployedVersion.ParentSequence)
	if err != nil {
		deployError = errors.Wrap(err, "failed to get app version archive")
		return deployError
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(deployedVersionArchive)
	if err != nil {
		deployError = errors.Wrap(err, "failed to load kotskinds")
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
			previouslyDeployedVersionArchive, err := store.GetStore().GetAppVersionArchive(a.ID, previouslyDeployedParentSequence)
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

	c, err := s.Server.GetChannel(clusterSocket.SocketID)
	if err != nil {
		return errors.Wrap(err, "failed to get socket channel from server")
	}
	c.Emit("deploy", deployArgs)
	clusterSocket.LastDeployedSequences[a.ID] = deployedVersion.ParentSequence

	// deploy status informers
	if len(kotsKinds.KotsApplication.Spec.StatusInformers) > 0 {
		registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
		if err != nil {
			return errors.Wrap(err, "failed to get registry settings for app")
		}

		// render status informers
		renderedInformers := []string{}
		for _, informer := range kotsKinds.KotsApplication.Spec.StatusInformers {
			renderedInformer, err := render.RenderContent(kotsKinds, registrySettings, deployedVersion.Sequence, a.IsAirgap, []byte(informer))
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to render status informer"))
				continue
			}
			if len(renderedInformer) == 0 {
				continue
			}
			renderedInformers = append(renderedInformers, string(renderedInformer))
		}

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

func (s *SocketService) supportBundleLoop() {
	for _, clusterSocket := range s.clusterSocketHistory {
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
			if err := s.processSupportBundle(clusterSocket, *sb); err != nil {
				logger.Error(errors.Wrapf(err, "failed to process support bundle %s for app %s", sb.ID, sb.AppID))
				continue
			}
		}
	}
}

func (s *SocketService) processSupportBundle(clusterSocket ClusterSocket, pendingSupportBundle supportbundletypes.PendingSupportBundle) error {
	a, err := store.GetStore().GetApp(pendingSupportBundle.AppID)
	if err != nil {
		return errors.Wrapf(err, "failed to get app %s", pendingSupportBundle.AppID)
	}

	c, err := s.Server.GetChannel(clusterSocket.SocketID)
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

	archivePath, err := store.GetStore().GetAppVersionArchive(a.ID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to get current archive")
	}
	defer os.RemoveAll(archivePath)

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to load current kotskinds")
	}

	err = supportbundle.CreateRenderedSpec(a.ID, sequence, "", true, kotsKinds.SupportBundle)
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

func (s *SocketService) restoreLoop() {
	for _, clusterSocket := range s.clusterSocketHistory {
		apps, err := store.GetStore().ListAppsForDownstream(clusterSocket.ClusterID)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to list installed apps for downstream"))
			continue
		}

		for _, a := range apps {
			if err := s.processRestoreForApp(clusterSocket, a); err != nil {
				logger.Error(errors.Wrapf(err, "failed to handle restoe for app %s", a.ID))
				continue
			}
		}
	}
}

func (s *SocketService) processRestoreForApp(clusterSocket ClusterSocket, a *apptypes.App) error {
	if a.RestoreInProgressName == "" {
		return nil
	}

	switch a.RestoreUndeployStatus {
	case apptypes.UndeployInProcess:
		// no-op

	case apptypes.UndeployCompleted:
		if err := handleUndeployCompleted(a); err != nil {
			return errors.Wrap(err, "failed to handle undeploy completed")
		}

	case apptypes.UndeployFailed:
		// no-op

	default:
		d, err := store.GetStore().GetDownstream(clusterSocket.ClusterID)
		if err != nil {
			return errors.Wrap(err, "failed to get downstream")
		}

		if err := s.undeployApp(a, d, clusterSocket); err != nil {
			return errors.Wrap(err, "failed to undeploy app")
		}
	}

	return nil
}

func handleUndeployCompleted(a *apptypes.App) error {
	restore, err := snapshot.GetRestore(a.RestoreInProgressName)
	if err != nil {
		return errors.Wrap(err, "failed to get restore")
	}

	if restore == nil {
		return errors.Wrap(startVeleroRestore(a.RestoreInProgressName), "failed to start velero restore")
	}

	return errors.Wrap(checkRestoreComplete(a, restore), "failed to check restore comlete")
}

func startVeleroRestore(restoreName string) error {
	logger.Info(fmt.Sprintf("creating velero restore object %s", restoreName))

	if err := snapshot.CreateRestore(restoreName); err != nil {
		return errors.Wrap(err, "failed to create restore")
	}

	return nil
}

func checkRestoreComplete(a *apptypes.App, restore *velerov1.Restore) error {
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

		sequenceStr, ok := backupAnnotations["kots.io/app-sequence"]
		if !ok || sequenceStr == "" {
			return errors.New("backup is missing sequence annotation")
		}

		sequence, err := strconv.ParseInt(sequenceStr, 10, 64)
		if err != nil {
			return errors.Wrap(err, "failed to parse sequence")
		}

		logger.Info(fmt.Sprintf("restore complete, setting deploy version to %d", sequence))

		if err := version.DeployVersion(a.ID, sequence); err != nil {
			return errors.Wrap(err, "failed to deploy version")
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
	archivePath, err := store.GetStore().GetAppVersionArchive(appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to get current archive")
	}
	defer os.RemoveAll(archivePath)

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to load current kotskinds")
	}

	err = supportbundle.CreateRenderedSpec(appID, sequence, origin, inCluster, kotsKinds.SupportBundle)
	if err != nil {
		return errors.Wrap(err, "failed to create rendered support bundle spec")
	}

	return nil
}

func (s *SocketService) undeployApp(a *apptypes.App, d *downstreamtypes.Downstream, clusterSocket ClusterSocket) error {
	deployedVersion, err := downstream.GetCurrentVersion(a.ID, d.ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current downstream version")
	}

	deployedVersionArchive, err := store.GetStore().GetAppVersionArchive(a.ID, deployedVersion.ParentSequence)
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

	c, err := s.Server.GetChannel(clusterSocket.SocketID)
	if err != nil {
		return errors.Wrap(err, "failed to get socket channel from server")
	}
	c.Emit("deploy", args)

	if err := app.SetRestoreUndeployStatus(a.ID, apptypes.UndeployInProcess); err != nil {
		return errors.Wrap(err, "failed to set restore undeploy status")
	}

	return nil
}
