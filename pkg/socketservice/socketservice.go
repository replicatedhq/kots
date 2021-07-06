package socketservice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	appstatustypes "github.com/replicatedhq/kots/pkg/api/appstatus/types"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/app"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	identitytypes "github.com/replicatedhq/kots/pkg/identity/types"
	snapshot "github.com/replicatedhq/kots/pkg/kotsadmsnapshot"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/redact"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/socket"
	"github.com/replicatedhq/kots/pkg/socket/transport"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterSocket struct {
	ClusterID             string
	SocketID              string
	LastDeployedSequences map[string]int64
}

type DeployArgs struct {
	AppID                string                `json:"app_id"`
	AppSlug              string                `json:"app_slug"`
	KubectlVersion       string                `json:"kubectl_version"`
	AdditionalNamespaces []string              `json:"additional_namespaces"`
	ImagePullSecret      string                `json:"image_pull_secret"`
	Namespace            string                `json:"namespace"`
	PreviousManifests    string                `json:"previous_manifests"`
	Manifests            string                `json:"manifests"`
	PreviousCharts       []byte                `json:"previous_charts"`
	Charts               []byte                `json:"charts"`
	Wait                 bool                  `json:"wait"`
	ResultCallback       string                `json:"result_callback"`
	ClearNamespaces      []string              `json:"clear_namespaces"`
	ClearPVCs            bool                  `json:"clear_pvcs"`
	AnnotateSlug         bool                  `json:"annotate_slug"`
	IsRestore            bool                  `json:"is_restore"`
	RestoreLabelSelector *metav1.LabelSelector `json:"restore_label_selector"`
}

type AppInformersArgs struct {
	AppID     string   `json:"app_id"`
	Informers []string `json:"informers"`
	Sequence  int64    `json:"sequence"`
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

// appDeployLoopErrorBackoff is a global map of loggers for each app that deploy loop uses to keep
// track of last time an error was logged and prevent duplicate logging.
var appDeployLoopErrorBackoff = map[string]*util.ErrorBackoff{}

func deployLoop() {
	for _, clusterSocket := range clusterSocketHistory {
		apps, err := store.GetStore().ListAppsForDownstream(clusterSocket.ClusterID)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to list installed apps for downstream"))
			continue
		}

		for _, a := range apps {
			deployed, err := processDeploySocketForApp(clusterSocket, a)
			if err != nil {
				_, ok := appDeployLoopErrorBackoff[a.ID]
				if !ok {
					appDeployLoopErrorBackoff[a.ID] = &util.ErrorBackoff{MinPeriod: 1 * time.Second, MaxPeriod: 30 * time.Minute}
				}
				appDeployLoopErrorBackoff[a.ID].OnError(err, func() {
					logger.Error(errors.Wrapf(err, "failed to run deploy loop for app %s in cluster %s", a.ID, clusterSocket.ClusterID))
				})
			} else if deployed {
				logger.Infof("Deploy success for app %s in cluster %s", a.ID, clusterSocket.ClusterID)
			}
		}
	}
}

func processDeploySocketForApp(clusterSocket *ClusterSocket, a *apptypes.App) (bool, error) {
	if a.RestoreInProgressName != "" {
		return false, nil
	}

	deployedVersion, err := store.GetStore().GetCurrentVersion(a.ID, clusterSocket.ClusterID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get current downstream version")
	} else if deployedVersion == nil {
		return false, nil
	}

	if value, ok := clusterSocket.LastDeployedSequences[a.ID]; ok && value == deployedVersion.ParentSequence {
		// this version is already the currently deployed version
		return false, nil
	}

	if err := deployVersionForApp(clusterSocket, a, deployedVersion); err != nil {
		return false, errors.Wrap(err, "failed to deploy version")
	}
	return true, nil
}

func kustomizeCharts(deployedVersionArchive string, name string, version string) ([]byte, error) {
	archive := []byte{}
	archiveChartDir := filepath.Join(deployedVersionArchive, "overlays", "downstreams", name, "charts")
	_, err := os.Stat(archiveChartDir)
	if err != nil {
		if os.IsNotExist(err) {
			return archive, nil
		}
		return archive, err
	}

	exportChartPath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return archive, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(exportChartPath)
	exportChartsDir := filepath.Join(exportChartPath, "charts")
	if _, err := os.Stat(exportChartsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(exportChartsDir, 0744); err != nil {
			return archive, errors.Wrap(err, "failed to mkdir for archive chart")
		}
	}

	baseDir := filepath.Join(deployedVersionArchive, "base", "charts")
	baseFiles := []string{"Chart.yaml", "Chart.lock"}

	err = filepath.Walk(archiveChartDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(archiveChartDir, filepath.Dir(path))
			if err != nil {
				return err
			}
			for _, filename := range baseFiles {
				err = exportFile(baseDir, rel, exportChartsDir, filename)
				if err != nil {
					return errors.Wrapf(err, "failed to export file %s", filename)
				}
			}
			if info.Name() == "kustomization.yaml" {
				archiveChartOutput, err := exec.Command(fmt.Sprintf("kustomize%s", version), "build", filepath.Dir(path)).Output()
				if err != nil {
					if ee, ok := err.(*exec.ExitError); ok {
						err = fmt.Errorf("kustomize %s: %q", path, string(ee.Stderr))
					}
					return errors.Wrapf(err, "failed to kustomize %s", path)
				}
				err = exportContent(archiveChartOutput, rel, exportChartsDir, "all.yaml")
				if err != nil {
					return errors.Wrapf(err, "failed to export content for %s", path)
				}
			}
			return nil
		})
	if err != nil {
		return archive, err
	}

	tempDir, err := ioutil.TempDir("", "helmkots")
	if err != nil {
		return archive, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tempDir)
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}
	if err := tarGz.Archive([]string{exportChartsDir}, path.Join(tempDir, "helmcharts.tar.gz")); err != nil {
		return archive, errors.Wrap(err, "failed to create tar gz")
	}
	archive, err = ioutil.ReadFile(path.Join(tempDir, "helmcharts.tar.gz"))
	if err != nil {
		return archive, errors.Wrap(err, "failed to read helm tar.gz file")
	}
	return archive, nil
}

func exportContent(allContent []byte, rel string, exportChartsDir string, filename string) error {
	relDir := ""
	// TODO: needs a comment explaining this
	if filepath.Base(rel) != "crds" {
		relDir = filepath.Join(exportChartsDir, rel, "templates")
	} else {
		relDir = filepath.Join(exportChartsDir, rel)
	}

	if err := os.MkdirAll(relDir, 0744); err != nil {
		return errors.Wrapf(err, "failed to mkdir for export chart %s", relDir)
	}
	exportFile := filepath.Join(relDir, filename)
	err := ioutil.WriteFile(exportFile, allContent, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to write file %s", exportFile)
	}
	return nil
}

func exportFile(baseDir string, rel string, exportChartsDir string, filename string) error {
	if filename != "Chart.yaml" && filename != "Chart.lock" {
		return nil
	}

	baseFile := filepath.Join(baseDir, rel, filename)
	baseFileContent, err := ioutil.ReadFile(baseFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "failed to read file from base")
	}

	relDir := filepath.Join(exportChartsDir, rel)
	if err := os.MkdirAll(relDir, 0744); err != nil {
		return errors.Wrap(err, "failed to create export chart dir")
	}

	exportFile := filepath.Join(relDir, filename)
	err = ioutil.WriteFile(exportFile, baseFileContent, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write file to export dir")
	}

	return nil
}

func deployVersionForApp(clusterSocket *ClusterSocket, a *apptypes.App, deployedVersion *downstreamtypes.DownstreamVersion) error {
	d, err := store.GetStore().GetDownstream(clusterSocket.ClusterID)
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

	builder, err := render.NewBuilder(kotsKinds, registrySettings, a.Slug, deployedVersion.Sequence, a.IsAirgap, os.Getenv("POD_NAMESPACE"))
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
	// Run kustomization on the charts as well
	deployedChartArchive, err := kustomizeCharts(deployedVersionArchive, d.Name, kotsKinds.KustomizeVersion())
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
	previouslyDeployedSequence, err := store.GetStore().GetPreviouslyDeployedSequence(a.ID, clusterSocket.ClusterID)
	if err != nil {
		deployError = errors.Wrap(err, "failed to get previously deployed sequence")
		return deployError
	}
	if previouslyDeployedSequence != -1 {
		previouslyDeployedParentSequence, err := store.GetStore().GetParentSequenceForSequence(a.ID, clusterSocket.ClusterID, previouslyDeployedSequence)
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
			previouslyDeployedChartArchive, err = kustomizeCharts(previouslyDeployedVersionArchive, d.Name, kotsKinds.KustomizeVersion())
			if err != nil {
				deployError = errors.Wrap(err, "failed to run kustomize on previously deployed charts")
				return deployError
			}

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
		Charts:               deployedChartArchive,
		PreviousCharts:       previouslyDeployedChartArchive,
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
			Sequence:  deployedVersion.Sequence,
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

		err := store.GetStore().SetAppStatus(a.ID, defaultReadyState, time.Now(), deployedVersion.Sequence)
		if err != nil {
			return errors.Wrap(err, "failed to set app status")
		}
		go reporting.SendAppInfo(a.ID)
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

		if err := undeployApp(a, d, clusterSocket, true); err != nil {
			return errors.Wrap(err, "failed to undeploy app")
		}
		break
	}

	return nil
}

func handleUndeployCompleted(clusterSocket *ClusterSocket, a *apptypes.App) error {
	snapshotName := a.RestoreInProgressName
	restoreName := a.RestoreInProgressName

	backup, err := snapshot.GetBackup(context.Background(), os.Getenv("POD_NAMESPACE"), snapshotName)
	if err != nil {
		return errors.Wrap(err, "failed to get backup")
	}
	if backup.Annotations["kots.io/instance"] == "true" {
		restoreName = fmt.Sprintf("%s.%s", snapshotName, a.Slug)
	}

	restore, err := snapshot.GetRestore(context.Background(), os.Getenv("POD_NAMESPACE"), restoreName)
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

	if err := snapshot.CreateApplicationRestore(context.Background(), os.Getenv("POD_NAMESPACE"), snapshotName, appSlug); err != nil {
		return errors.Wrap(err, "failed to create restore")
	}

	return nil
}

func checkRestoreComplete(clusterSocket *ClusterSocket, a *apptypes.App, restore *velerov1.Restore) error {
	switch restore.Status.Phase {
	case velerov1.RestorePhaseCompleted:
		backup, err := snapshot.GetBackup(context.Background(), os.Getenv("POD_NAMESPACE"), restore.Spec.BackupName)
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

		// mark the sequence as deployed both in the db and socket history
		// so that the admin console does not try to re-deploy it
		if err := version.DeployVersion(a.ID, sequence); err != nil {
			return errors.Wrap(err, "failed to mark app version as deployed")
		}
		socketMtx.Lock()
		clusterSocket.LastDeployedSequences[a.ID] = sequence
		socketMtx.Unlock()

		if err := createSupportBundleSpec(a.ID, sequence, "", true); err != nil {
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

func createSupportBundleSpec(appID string, sequence int64, origin string, inCluster bool) error {
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

	defaultOpts := supportbundle.DefaultTroubleshootOpts{
		Origin:    origin,
		InCluster: inCluster,
	}

	_, err = supportbundle.CreateRenderedSpec(appID, sequence, kotsKinds, defaultOpts)
	if err != nil {
		return errors.Wrap(err, "failed to create rendered support bundle spec")
	}

	err = redact.CreateRenderedAppRedactSpec(appID, sequence, kotsKinds)
	if err != nil {
		return errors.Wrap(err, "failed to write app redact spec configmap")
	}

	return nil
}

func undeployApp(a *apptypes.App, d *downstreamtypes.Downstream, clusterSocket *ClusterSocket, isRestore bool) error {
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

	backup, err := snapshot.GetBackup(context.Background(), os.Getenv("POD_NAMESPACE"), a.RestoreInProgressName)
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

	args := DeployArgs{
		AppID:                a.ID,
		AppSlug:              a.Slug,
		KubectlVersion:       kotsKinds.KotsApplication.Spec.KubectlVersion,
		Namespace:            ".",
		Manifests:            "",
		PreviousManifests:    base64EncodedManifests,
		ResultCallback:       "/api/v1/undeploy/result",
		Wait:                 true,
		ClearNamespaces:      backup.Spec.IncludedNamespaces,
		ClearPVCs:            true,
		IsRestore:            isRestore,
		RestoreLabelSelector: restoreLabelSelector,
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
