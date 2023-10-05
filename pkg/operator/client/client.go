package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/mholt/archiver/v3"
	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/app"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/appstate"
	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	"github.com/replicatedhq/kots/pkg/binaries"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/operator/applier"
	operatortypes "github.com/replicatedhq/kots/pkg/operator/types"
	"github.com/replicatedhq/kots/pkg/registry"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	supportbundletypes "github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kotskinds/pkg/helmchart"
	"go.uber.org/zap"
)

type DeployResults struct {
	IsError      bool   `json:"isError"`
	DryrunStdout []byte `json:"dryrunStdout"`
	DryrunStderr []byte `json:"dryrunStderr"`
	ApplyStdout  []byte `json:"applyStdout"`
	ApplyStderr  []byte `json:"applyStderr"`
	HelmStdout   []byte `json:"helmStdout"`
	HelmStderr   []byte `json:"helmStderr"`
}

// DesiredState is what we receive from the kotsadm api server
type DesiredState struct {
	Present []operatortypes.DeployAppArgs `json:"present"`
	Missing map[string][]string           `json:"missing"`
}

type Client struct {
	TargetNamespace string

	watchedNamespaces []string
	imagePullSecrets  []string

	appStateMonitor       *appstate.Monitor
	HookStopChans         []chan struct{}
	namespaceStopChan     chan struct{}
	ExistingHookInformers map[string]bool // namespaces map to invoke the Informer once during deploy
}

func (c *Client) Init() error {
	if _, ok := c.ExistingHookInformers[c.TargetNamespace]; !ok {
		c.ExistingHookInformers[c.TargetNamespace] = true
		if err := c.runHooksInformer(c.TargetNamespace); err != nil {
			// we don't fail here...
			log.Printf("error registering cleanup hooks for TargetNamespace: %s: %s", c.TargetNamespace, err.Error())
		}
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	c.appStateMonitor = appstate.NewMonitor(clientset, c.TargetNamespace)
	go c.runAppStateMonitor()

	return nil
}

func (c *Client) Shutdown() {
	log.Println("Shutting down the operator client")

	c.shutdownHooksInformer()
	c.shutdownNamespacesInformer()

	if c.appStateMonitor != nil {
		c.appStateMonitor.Shutdown()
	}
}

func (c *Client) runAppStateMonitor() error {
	m := map[string]func(f func()){}
	hash := map[string]uint64{}
	var mtx sync.Mutex

	for appStatus := range c.appStateMonitor.AppStatusChan() {
		throttled, ok := m[appStatus.AppID]
		if !ok {
			throttled = util.NewThrottle(time.Second)
			m[appStatus.AppID] = throttled
		}
		throttled(func() {
			mtx.Lock()
			lastHash := hash[appStatus.AppID]
			nextHash, _ := hashstructure.Hash(appStatus, nil)
			hash[appStatus.AppID] = nextHash
			mtx.Unlock()
			if lastHash != nextHash {
				b, _ := json.Marshal(appStatus)
				log.Printf("Updating app status %s", b)
			}
			if err := c.setAppStatus(appStatus); err != nil {
				log.Printf("error updating app status: %v", err)
			}
		})
	}

	return errors.New("app state monitor shutdown")
}

func (c *Client) ApplyNamespacesInformer(namespaces []string, imagePullSecrets []string) {
	for _, ns := range namespaces {
		if ns == "*" {
			continue
		}
		if err := c.ensureNamespacePresent(ns); err != nil {
			// we don't fail here...
			log.Printf("error creating namespace: %s", err.Error())
		}
		if err := c.ensureImagePullSecretsPresent(ns, imagePullSecrets); err != nil {
			// we don't fail here...
			log.Printf("error ensuring image pull secrets for namespace %s: %s", ns, err.Error())
		}
	}

	c.imagePullSecrets = imagePullSecrets
	c.watchedNamespaces = namespaces

	c.shutdownNamespacesInformer()
	if len(c.watchedNamespaces) > 0 {
		c.runNamespacesInformer()
	}
}

func (c *Client) ApplyHooksInformer(namespaces []string) {
	for _, ns := range namespaces {
		if ns == "*" {
			continue
		}
		if _, ok := c.ExistingHookInformers[ns]; !ok {
			c.ExistingHookInformers[ns] = true
			if err := c.runHooksInformer(ns); err != nil {
				// we don't fail here...
				log.Printf("error registering cleanup hooks for namespace: %s: %s", ns, err.Error())
			}
		}
	}
}

func (c *Client) DeployApp(deployArgs operatortypes.DeployAppArgs) (deployed bool, finalError error) {
	log.Println("received a deploy request for", deployArgs.AppSlug)

	var deployRes *deployResult
	var helmResult *commandResult
	var deployError, helmError error

	defer func() {
		results, err := c.setDeployResults(deployArgs, &deployRes.dryRunResult, &deployRes.applyResult, helmResult)
		if err != nil {
			finalError = errors.Wrap(err, "failed to set results")
		}
		if results != nil {
			deployed = !results.IsError
		}
	}()

	deployRes, deployError = c.deployManifests(deployArgs)
	if deployError != nil {
		deployRes = &deployResult{}
		deployRes.applyResult.hasErr = true
		deployRes.applyResult.multiStderr = [][]byte{[]byte(deployError.Error())}
		log.Printf("failed to deploy manifests: %v", deployError)
		return
	}

	helmResult, helmError = c.deployHelmCharts(deployArgs)
	if helmError != nil {
		helmResult = &commandResult{}
		helmResult.hasErr = true
		helmResult.multiStderr = [][]byte{[]byte(helmError.Error())}
		log.Printf("failed to deploy helm charts: %v", helmError)
		return
	}

	return
}

func (c *Client) UndeployApp(undeployArgs operatortypes.UndeployAppArgs) (finalError error) {
	log.Println("received an undeploy request for", undeployArgs.AppSlug)

	defer func() {
		var status apptypes.UndeployStatus
		if finalError != nil {
			status = apptypes.UndeployFailed
		} else {
			status = apptypes.UndeployCompleted
		}
		if err := c.setUndeployStatus(undeployArgs, status); err != nil {
			finalError = errors.Wrap(err, "failed to set undeploy status")
			log.Printf("failed to set undeploy status: %v", err)
		}
	}()

	if err := c.undeployHelmCharts(undeployArgs); err != nil {
		log.Printf("failed to undeploy helm charts: %v", err)
		return errors.Wrap(err, "failed to undeploy helm charts")
	}

	if err := c.undeployManifests(undeployArgs); err != nil {
		log.Printf("failed to undeploy manifests: %v", err)
		return errors.Wrap(err, "failed to undeploy manifests")
	}

	return nil
}

func (c *Client) deployManifests(deployArgs operatortypes.DeployAppArgs) (*deployResult, error) {
	if deployArgs.PreviousManifests != "" {
		opts := DiffAndDeleteOptions{
			PreviousManifests:    deployArgs.PreviousManifests,
			CurrentManifests:     deployArgs.Manifests,
			AdditionalNamespaces: deployArgs.AdditionalNamespaces,
			IsRestore:            deployArgs.IsRestore,
			RestoreLabelSelector: deployArgs.RestoreLabelSelector,
			KubectlVersion:       deployArgs.KubectlVersion,
			KustomizeVersion:     deployArgs.KustomizeVersion,
			Wait:                 deployArgs.Wait,
		}
		if err := c.diffAndDeleteManifests(opts); err != nil {
			return nil, errors.Wrapf(err, "failed to diff and delete manifests")
		}
	}

	result, err := c.ensureResourcesPresent(deployArgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deploy")
	}

	return result, nil
}

func (c *Client) deployHelmCharts(deployArgs operatortypes.DeployAppArgs) (*commandResult, error) {
	// extract previous v1beta1 helm charts
	prevV1Beta1HelmDir, err := extractHelmCharts(deployArgs.PreviousV1Beta1ChartsArchive, "prev-v1beta1")
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract previous helm charts")
	}
	defer os.RemoveAll(prevV1Beta1HelmDir)

	// extract current v1beta1 helm charts
	curV1Beta1HelmDir, err := extractHelmCharts(deployArgs.V1Beta1ChartsArchive, "curr-v1beta1")
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract current helm charts")
	}
	defer os.RemoveAll(curV1Beta1HelmDir)

	// extract previous v1beta2 helm charts
	prevV1Beta2HelmDir, err := extractHelmCharts(deployArgs.PreviousV1Beta2ChartsArchive, "prev-v1beta2")
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract previous helm charts")
	}
	defer os.RemoveAll(prevV1Beta2HelmDir)

	// extract current v1beta2 helm charts
	curV1Beta2HelmDir, err := extractHelmCharts(deployArgs.V1Beta2ChartsArchive, "curr-v1beta2")
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract current helm charts")
	}
	defer os.RemoveAll(curV1Beta2HelmDir)

	// find removed charts
	prevKotsV1Beta1Charts := []helmchart.HelmChartInterface{}
	if deployArgs.PreviousKotsKinds != nil && deployArgs.PreviousKotsKinds.V1Beta1HelmCharts != nil {
		for _, kotsChart := range deployArgs.PreviousKotsKinds.V1Beta1HelmCharts.Items {
			kc := kotsChart
			prevKotsV1Beta1Charts = append(prevKotsV1Beta1Charts, &kc)
		}
	}

	curV1Beta1KotsCharts := []helmchart.HelmChartInterface{}
	if deployArgs.KotsKinds != nil && deployArgs.KotsKinds.V1Beta1HelmCharts != nil {
		for _, kotsChart := range deployArgs.KotsKinds.V1Beta1HelmCharts.Items {
			kc := kotsChart
			curV1Beta1KotsCharts = append(curV1Beta1KotsCharts, &kc)
		}
	}

	prevKotsV1Beta2Charts := []helmchart.HelmChartInterface{}
	if deployArgs.PreviousKotsKinds != nil && deployArgs.PreviousKotsKinds.V1Beta2HelmCharts != nil {
		for _, kotsChart := range deployArgs.PreviousKotsKinds.V1Beta2HelmCharts.Items {
			kc := kotsChart
			prevKotsV1Beta2Charts = append(prevKotsV1Beta2Charts, &kc)
		}
	}

	curV1Beta2KotsCharts := []helmchart.HelmChartInterface{}
	if deployArgs.KotsKinds != nil && deployArgs.KotsKinds.V1Beta2HelmCharts != nil {
		for _, kotsChart := range deployArgs.KotsKinds.V1Beta2HelmCharts.Items {
			kc := kotsChart
			curV1Beta2KotsCharts = append(curV1Beta2KotsCharts, &kc)
		}
	}

	opts := getRemovedChartsOptions{
		prevV1Beta1Dir:            prevV1Beta1HelmDir,
		curV1Beta1Dir:             curV1Beta1HelmDir,
		previousV1Beta1KotsCharts: prevKotsV1Beta1Charts,
		currentV1Beta1KotsCharts:  curV1Beta1KotsCharts,
		prevV1Beta2Dir:            prevV1Beta2HelmDir,
		curV1Beta2Dir:             curV1Beta2HelmDir,
		previousV1Beta2KotsCharts: prevKotsV1Beta2Charts,
		currentV1Beta2KotsCharts:  curV1Beta2KotsCharts,
	}
	removedCharts, err := getRemovedCharts(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find removed charts")
	}

	// uninstall removed charts
	if len(removedCharts) > 0 {
		v1Beta1ChartsDir := ""
		if prevV1Beta1HelmDir != "" {
			v1Beta1ChartsDir = filepath.Join(prevV1Beta1HelmDir, "charts")
		}
		v1Beta2ChartsDir := ""
		if prevV1Beta2HelmDir != "" {
			v1Beta2ChartsDir = filepath.Join(prevV1Beta2HelmDir, "helm")
		}

		err := c.uninstallWithHelm(v1Beta1ChartsDir, v1Beta2ChartsDir, removedCharts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to uninstall helm charts")
		}
	}

	var installResult *commandResult
	// deploy current helm charts
	if len(deployArgs.V1Beta1ChartsArchive) > 0 || len(deployArgs.V1Beta2ChartsArchive) > 0 {
		kotsCharts := append(curV1Beta1KotsCharts, curV1Beta2KotsCharts...)

		v1Beta1ChartsDir := ""
		if curV1Beta1HelmDir != "" {
			v1Beta1ChartsDir = filepath.Join(curV1Beta1HelmDir, "charts")
		}

		v1Beta2ChartsDir := ""
		if curV1Beta2HelmDir != "" {
			v1Beta2ChartsDir = filepath.Join(curV1Beta2HelmDir, "helm")
		}

		installResult, err = c.installWithHelm(v1Beta1ChartsDir, v1Beta2ChartsDir, kotsCharts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to install helm charts")
		}
	}

	return installResult, nil
}

// extractHelmCharts extracts the helm charts from the archive and returns the path to the directory.
// If the archive is empty, an empty string is returned.
func extractHelmCharts(chartsArchive []byte, dirName string) (helmDir string, err error) {
	if len(chartsArchive) == 0 {
		return "", nil
	}

	tmpDir, err := os.MkdirTemp("", "helm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir for previous charts")
	}

	err = ioutil.WriteFile(path.Join(tmpDir, "archive.tar.gz"), chartsArchive, 0644)
	if err != nil {
		return "", errors.Wrap(err, "failed to write previous archive")
	}

	helmDir = path.Join(tmpDir, dirName)
	if err := os.MkdirAll(helmDir, 0755); err != nil {
		return "", errors.Wrap(err, "failed to create dir to stage previous helm archive")
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}

	if err := tarGz.Unarchive(path.Join(tmpDir, "archive.tar.gz"), helmDir); err != nil {
		return "", errors.Wrap(err, "falied to unarchive previous helm archive")
	}

	return helmDir, nil
}

func (c *Client) undeployManifests(undeployArgs operatortypes.UndeployAppArgs) error {
	if undeployArgs.Manifests != "" {
		opts := DiffAndDeleteOptions{
			PreviousManifests:    undeployArgs.Manifests,
			CurrentManifests:     "",
			AdditionalNamespaces: undeployArgs.AdditionalNamespaces,
			IsRestore:            undeployArgs.IsRestore,
			RestoreLabelSelector: undeployArgs.RestoreLabelSelector,
			KubectlVersion:       undeployArgs.KubectlVersion,
			KustomizeVersion:     undeployArgs.KustomizeVersion,
			Wait:                 undeployArgs.Wait,
		}
		if err := c.diffAndDeleteManifests(opts); err != nil {
			return errors.Wrapf(err, "failed to diff and delete manifests")
		}
	}

	if undeployArgs.ClearPVCs {
		// TODO: multi-namespace support
		err := c.deletePVCs(undeployArgs.RestoreLabelSelector, undeployArgs.AppSlug)
		if err != nil {
			return errors.Wrap(err, "failed to delete PVCs")
		}
	}

	if len(undeployArgs.ClearNamespaces) > 0 {
		err := c.clearNamespaces(undeployArgs.AppSlug, undeployArgs.ClearNamespaces, undeployArgs.IsRestore, undeployArgs.RestoreLabelSelector)
		if err != nil {
			return errors.Wrap(err, "failed to clear namespaces")
		}
	}

	// TODO: delete the additional namespaces if they are empty

	return nil
}

func (c *Client) undeployHelmCharts(undeployArgs operatortypes.UndeployAppArgs) error {
	kotsCharts := []helmchart.HelmChartInterface{}
	if undeployArgs.KotsKinds != nil {
		if undeployArgs.KotsKinds.V1Beta1HelmCharts != nil {
			for _, v1Beta1Chart := range undeployArgs.KotsKinds.V1Beta1HelmCharts.Items {
				kc := v1Beta1Chart
				kotsCharts = append(kotsCharts, &kc)
			}
		}
		if undeployArgs.KotsKinds.V1Beta2HelmCharts != nil {
			for _, v1Beta2Chart := range undeployArgs.KotsKinds.V1Beta2HelmCharts.Items {
				kc := v1Beta2Chart
				kotsCharts = append(kotsCharts, &kc)
			}
		}
	}

	v1Beta1HelmDir, err := extractHelmCharts(undeployArgs.V1Beta1ChartsArchive, "v1beta1")
	if err != nil {
		return errors.Wrap(err, "failed to extract v1beta1 helm charts")
	}
	defer os.RemoveAll(v1Beta1HelmDir)

	v1Beta2HelmDir, err := extractHelmCharts(undeployArgs.V1Beta2ChartsArchive, "v1beta2")
	if err != nil {
		return errors.Wrap(err, "failed to extract v1beta2 helm charts")
	}
	defer os.RemoveAll(v1Beta2HelmDir)

	v1Beta1ChartsDir := ""
	if v1Beta1HelmDir != "" {
		v1Beta1ChartsDir = filepath.Join(v1Beta1HelmDir, "charts")
	}

	v1Beta2ChartsDir := ""
	if v1Beta2HelmDir != "" {
		v1Beta2ChartsDir = filepath.Join(v1Beta2HelmDir, "helm")
	}

	if err := c.uninstallWithHelm(v1Beta1ChartsDir, v1Beta2ChartsDir, kotsCharts); err != nil {
		return errors.Wrap(err, "failed to uninstall helm charts")
	}

	return nil
}

func (c *Client) setDeployResults(args operatortypes.DeployAppArgs, dryRunResult *commandResult, applyResult *commandResult, helmResult *commandResult) (*DeployResults, error) {
	results := &DeployResults{}

	if dryRunResult != nil {
		results.IsError = results.IsError || dryRunResult.hasErr
		results.DryrunStdout = bytes.Join(dryRunResult.multiStdout, []byte("\n"))
		results.DryrunStderr = bytes.Join(dryRunResult.multiStderr, []byte("\n"))
	}

	if applyResult != nil {
		results.IsError = results.IsError || applyResult.hasErr
		results.ApplyStdout = bytes.Join(applyResult.multiStdout, []byte("\n"))
		results.ApplyStderr = bytes.Join(applyResult.multiStderr, []byte("\n"))
	}

	if helmResult != nil {
		results.IsError = results.IsError || helmResult.hasErr
		results.HelmStdout = bytes.Join(helmResult.multiStdout, []byte("\n"))
		results.HelmStderr = bytes.Join(helmResult.multiStderr, []byte("\n"))
	}

	app, err := store.GetStore().GetApp(args.AppID)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get app after deploying"))
	} else {
		troubleshootOpts := supportbundletypes.TroubleshootOptions{
			InCluster: true,
		}
		if _, err := supportbundle.CreateSupportBundleDependencies(app, args.Sequence, troubleshootOpts); err != nil {
			// support bundle is not essential. keep processing deployment request
			logger.Error(errors.Wrapf(err, "failed to create support bundle for sequence %d after deploying", args.Sequence))
		}
	}

	alreadySuccessful, err := store.GetStore().IsDownstreamDeploySuccessful(args.AppID, args.ClusterID, args.Sequence)
	if err != nil {
		return results, errors.Wrap(err, "failed to check deploy successful")
	}

	if alreadySuccessful {
		return results, nil
	}

	downstreamOutput := downstreamtypes.DownstreamOutput{
		DryrunStdout: base64.StdEncoding.EncodeToString(results.DryrunStdout),
		DryrunStderr: base64.StdEncoding.EncodeToString(results.DryrunStderr),
		ApplyStdout:  base64.StdEncoding.EncodeToString(results.ApplyStdout),
		ApplyStderr:  base64.StdEncoding.EncodeToString(results.ApplyStderr),
		HelmStdout:   base64.StdEncoding.EncodeToString(results.HelmStdout),
		HelmStderr:   base64.StdEncoding.EncodeToString(results.HelmStderr),
		RenderError:  "",
	}
	err = store.GetStore().UpdateDownstreamDeployStatus(args.AppID, args.ClusterID, args.Sequence, results.IsError, downstreamOutput)
	if err != nil {
		return results, errors.Wrap(err, "failed to update downstream deploy status")
	}

	if !results.IsError {
		go func() {
			err := registry.DeleteUnusedImages(args.AppID, false)
			if err != nil {
				if _, ok := err.(registry.AppRollbackError); ok {
					logger.Infof("not garbage collecting images because version allows rollbacks: %v", err)
				} else {
					logger.Infof("failed to delete unused images: %v", err)
				}
			}
		}()
	}

	return results, nil
}

func (c *Client) setUndeployStatus(args operatortypes.UndeployAppArgs, status apptypes.UndeployStatus) error {
	logger.Info("undeploy status",
		zap.String("status", string(status)),
		zap.String("appID", args.AppID))

	foundApp, err := store.GetStore().GetApp(args.AppID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	if foundApp.RestoreInProgressName != "" {
		go func() {
			<-time.After(20 * time.Second)
			err = app.SetRestoreUndeployStatus(args.AppID, status)
			if err != nil {
				err = errors.Wrap(err, "failed to set app undeploy status")
				logger.Error(err)
				return
			}
		}()
	}

	return nil
}

func (c *Client) ApplyAppInformers(args operatortypes.AppInformersArgs) {
	log.Printf("received an inform event: %#v", args)

	appID := args.AppID
	sequence := args.Sequence
	informerStrings := args.Informers

	var informers []appstatetypes.StatusInformer
	for _, str := range informerStrings {
		informer, err := str.Parse()
		if err != nil {
			log.Printf("failed to parse informer %s: %s", str, err.Error())
			continue // don't stop
		}
		informers = append(informers, informer)
	}
	if len(informers) > 0 {
		c.appStateMonitor.Apply(appID, sequence, informers)
	}
}

func (c *Client) setAppStatus(newAppStatus appstatetypes.AppStatus) error {
	currentAppStatus, err := store.GetStore().GetAppStatus(newAppStatus.AppID)
	if err != nil {
		return errors.Wrap(err, "failed to get current app status")
	}

	err = store.GetStore().SetAppStatus(newAppStatus.AppID, newAppStatus.ResourceStates, newAppStatus.UpdatedAt, newAppStatus.Sequence)
	if err != nil {
		return errors.Wrap(err, "failed to set app status")
	}

	newAppState := appstatetypes.GetState(newAppStatus.ResourceStates)
	if currentAppStatus != nil && newAppState != currentAppStatus.State {
		go func() {
			err := reporting.GetReporter().SubmitAppInfo(newAppStatus.AppID)
			if err != nil {
				logger.Debugf("failed to submit app info: %v", err)
			}
		}()
	}

	return nil
}

func (c *Client) getApplier(kubectlVersion, kustomizeVersion string) (applier.KubectlInterface, error) {
	kubectl, err := binaries.GetKubectlPathForVersion(kubectlVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find kubectl")
	}

	kustomize, err := binaries.GetKustomizePathForVersion(kustomizeVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find kustomize")
	}

	config, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	return applier.NewKubectl(kubectl, kustomize, config), nil
}
