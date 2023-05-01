package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"

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

	appStateMonitor   *appstate.Monitor
	HookStopChans     []chan struct{}
	namespaceStopChan chan struct{}
	ExistingInformers map[string]bool // namespaces map to invoke the Informer once during deploy
}

func (c *Client) Init() error {
	if _, ok := c.ExistingInformers[c.TargetNamespace]; !ok {
		c.ExistingInformers[c.TargetNamespace] = true
		if err := c.runHooksInformer(c.TargetNamespace); err != nil {
			// we don't fail here...
			log.Printf("error registering cleanup hooks for TargetNamespace: %s: %s",
				c.TargetNamespace, err.Error())
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

	c.shutdownNamespacesInformer()
	if len(c.watchedNamespaces) > 0 {
		c.runNamespacesInformer()
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

	for _, additionalNamespace := range deployArgs.AdditionalNamespaces {
		if additionalNamespace == "*" {
			continue
		}
		if err := c.ensureNamespacePresent(additionalNamespace); err != nil {
			// we don't fail here...
			log.Printf("error creating namespace: %s", err.Error())
		}
		if err := c.ensureImagePullSecretsPresent(additionalNamespace, deployArgs.ImagePullSecrets); err != nil {
			// we don't fail here...
			log.Printf("error ensuring image pull secrets for namespace %s: %s", additionalNamespace, err.Error())
		}
		if _, ok := c.ExistingInformers[additionalNamespace]; !ok {
			c.ExistingInformers[additionalNamespace] = true
			if err := c.runHooksInformer(additionalNamespace); err != nil {
				// we don't fail here...
				log.Printf("error registering cleanup hooks for additionalNamespace: %s: %s",
					additionalNamespace, err.Error())
			}
		}
	}
	c.imagePullSecrets = deployArgs.ImagePullSecrets
	c.watchedNamespaces = deployArgs.AdditionalNamespaces

	result, err := c.ensureResourcesPresent(deployArgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deploy")
	}

	return result, nil
}

func (c *Client) deployHelmCharts(deployArgs operatortypes.DeployAppArgs) (*commandResult, error) {
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}

	var prevHelmDir string
	if len(deployArgs.PreviousCharts) > 0 {
		tmpDir, err := ioutil.TempDir("", "helm")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create temp dir for previous charts")
		}
		defer os.RemoveAll(tmpDir)

		err = ioutil.WriteFile(path.Join(tmpDir, "archive.tar.gz"), deployArgs.PreviousCharts, 0644)
		if err != nil {
			return nil, errors.Wrap(err, "failed to write previous archive")
		}

		prevHelmDir = path.Join(tmpDir, "prevhelm")
		if err := os.MkdirAll(prevHelmDir, 0755); err != nil {
			return nil, errors.Wrap(err, "failed to create dir to stage previous helm archive")
		}

		if err := tarGz.Unarchive(path.Join(tmpDir, "archive.tar.gz"), prevHelmDir); err != nil {
			return nil, errors.Wrap(err, "falied to unarchive previous helm archive")
		}
	}

	var curHelmDir string
	var installResult *commandResult
	if len(deployArgs.Charts) > 0 {
		tmpDir, err := ioutil.TempDir("", "helm")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create temp dir to stage currently deployed archive")
		}
		defer os.RemoveAll(tmpDir)

		err = ioutil.WriteFile(path.Join(tmpDir, "archive.tar.gz"), deployArgs.Charts, 0644)
		if err != nil {
			return nil, errors.Wrap(err, "failed to write current archive")
		}

		curHelmDir = path.Join(tmpDir, "currhelm")
		if err := os.MkdirAll(curHelmDir, 0755); err != nil {
			return nil, errors.Wrap(err, "failed to create dir to stage currently deployed archive")
		}

		if err := tarGz.Unarchive(path.Join(tmpDir, "archive.tar.gz"), curHelmDir); err != nil {
			return nil, errors.Wrap(err, "failed to unarchive current helm archive")
		}
	}

	curKotsCharts := []v1beta1.HelmChart{}
	if deployArgs.KotsKinds != nil && deployArgs.KotsKinds.HelmCharts != nil {
		curKotsCharts = deployArgs.KotsKinds.HelmCharts.Items
	}

	previousKotsCharts := []v1beta1.HelmChart{}
	if deployArgs.PreviousKotsKinds != nil && deployArgs.PreviousKotsKinds.HelmCharts != nil {
		previousKotsCharts = deployArgs.PreviousKotsKinds.HelmCharts.Items
	}

	removedCharts, err := getRemovedCharts(prevHelmDir, curHelmDir, previousKotsCharts, curKotsCharts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find removed charts")
	}
	if len(removedCharts) > 0 {
		err := c.uninstallWithHelm(prevHelmDir, removedCharts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to uninstall helm charts")
		}
	}

	if len(deployArgs.Charts) > 0 {
		installResult, err = c.installWithHelm(curHelmDir, curKotsCharts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to install helm charts")
		}
	}

	return installResult, nil
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
	if len(undeployArgs.Charts) > 0 {
		return nil
	}

	tmpDir, err := ioutil.TempDir("", "helm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir for charts")
	}
	defer os.RemoveAll(tmpDir)

	err = ioutil.WriteFile(path.Join(tmpDir, "archive.tar.gz"), undeployArgs.Charts, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write archive")
	}

	helmDir := path.Join(tmpDir, "prevhelm")
	if err := os.MkdirAll(helmDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create dir to stage helm archive")
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(path.Join(tmpDir, "archive.tar.gz"), helmDir); err != nil {
		return errors.Wrap(err, "falied to unarchive helm archive")
	}

	kotsCharts := []v1beta1.HelmChart{}
	if undeployArgs.KotsKinds != nil && undeployArgs.KotsKinds.HelmCharts != nil {
		kotsCharts = undeployArgs.KotsKinds.HelmCharts.Items
	}
	if err := c.uninstallWithHelm(helmDir, kotsCharts); err != nil {
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
