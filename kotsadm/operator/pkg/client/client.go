package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/mholt/archiver"
	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/operator/pkg/applier"
	"github.com/replicatedhq/kots/kotsadm/operator/pkg/appstate"
	"github.com/replicatedhq/kots/kotsadm/operator/pkg/appstate/types"
	"github.com/replicatedhq/kots/kotsadm/operator/pkg/socket"
	"github.com/replicatedhq/kots/kotsadm/operator/pkg/socket/transport"
	"github.com/replicatedhq/kots/kotsadm/operator/pkg/supportbundle"
	"github.com/replicatedhq/kots/kotsadm/operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	socketDeployMtxs = map[string]*sync.Mutex{} // key is app id
)

type ApplicationManifests struct {
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

// DesiredState is what we receive from the kotsadm-api server
type DesiredState struct {
	Present []ApplicationManifests `json:"present"`
	Missing map[string][]string    `json:"missing"`
}

type InformRequest struct {
	AppID     string                       `json:"app_id"`
	Sequence  int64                        `json:"sequence"`
	Informers []types.StatusInformerString `json:"informers"`
}

type Client struct {
	APIEndpoint     string
	Token           string
	TargetNamespace string

	watchedNamespaces []string
	imagePullSecret   string

	appStateMonitor   *appstate.Monitor
	HookStopChans     []chan struct{}
	namespaceStopChan chan struct{}
	ExistingInformers map[string]bool // namespaces map to invoke the Informer once during deploy
}

// Run is the main entrypoint of the operator when running in standard, normal operations
func (c *Client) Run() error {
	log.Println("Starting kotsadm-operator loop")

	supportbundle.StartServer()

	if _, ok := c.ExistingInformers[c.TargetNamespace]; !ok {
		c.ExistingInformers[c.TargetNamespace] = true
		if err := c.runHooksInformer(c.TargetNamespace); err != nil {
			// we don't fail here...
			log.Printf("error registering cleanup hooks for TargetNamespace: %s: %s",
				c.TargetNamespace, err.Error())
		}
	}

	defer c.shutdownHooksInformer()
	defer c.shutdownNamespacesInformer()

	for {
		err := c.connect()
		if err != nil {
			// this needs a backoff
			log.Printf("unable to connect to api: %v\n", err)
			time.Sleep(time.Second * 2)
			continue
		}

		// some easy backoff for now
		time.Sleep(time.Second * 2)
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
				log.Printf("Sending app status %s", b)
			}
			if err := c.sendAppStatus(appStatus); err != nil {
				log.Printf("error sending app status: %v", err)
			}
		})
	}

	return errors.New("app state monitor shutdown")
}

// connect will return an error on a fatal error, or nil if the server
// disconnected us or a network problem disconnected us
func (c *Client) connect() error {
	u, err := url.Parse(c.APIEndpoint)
	if err != nil {
		return errors.Wrap(err, "failed to parse url")
	}

	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return errors.Wrap(err, "failed to parse port")
	}

	hasConnected := false
	isUnexpectedlyDisconnected := false

	log.Printf("connecting to api at %s\n", c.APIEndpoint)
	socketClient := socket.NewClient()

	err = socketClient.On(socket.OnConnection, func(h *socket.Channel) {
		log.Println("received a connection event")
		hasConnected = true
	})
	if err != nil {
		return errors.Wrap(err, "failed to register connected handler")
	}

	err = socketClient.On(socket.OnDisconnection, func(h *socket.Channel, args interface{}) {
		log.Printf("received a disconnected event %#v", args)
		isUnexpectedlyDisconnected = true
	})
	if err != nil {
		return errors.Wrap(err, "failed to register disconnected handler")
	}

	if err := c.registerHandlers(socketClient); err != nil {
		return errors.Wrap(err, "failed to register handlers")
	}

	err = socketClient.Dial(socket.GetUrl(u.Hostname(), port, c.Token, false), transport.GetDefaultWebsocketTransport())
	if err != nil {
		return errors.Wrap(err, "failed to connect")
	}
	defer socketClient.Close()

	restconfig, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get in cluster config")
	}
	clientset, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return errors.Wrap(err, "failed to get new kubernetes client")
	}

	c.appStateMonitor = appstate.NewMonitor(clientset, c.TargetNamespace)
	defer c.appStateMonitor.Shutdown()

	go c.runAppStateMonitor()

	// wait for a connection for at least 2 seconds
	time.Sleep(time.Second * 2)
	if !hasConnected {
		log.Println("expected to be connected to the api by now, but it's not true. disappointing...  (will retry)")
		return nil // allow another attempt
	}

	for {
		if isUnexpectedlyDisconnected {
			log.Println("unexpectedly disconnected from api (will reconnect)")
			return nil
		}

		time.Sleep(time.Second)
	}
}

func (c *Client) registerHandlers(socketClient *socket.Client) error {
	var err error

	err = socketClient.On("deploy", func(h *socket.Channel, args ApplicationManifests) {
		// this mutex is mainly to prevent the app from being deployed and undeployed at the same time
		// or to prevent two app versions from being deployed at the same time
		if _, ok := socketDeployMtxs[args.AppID]; !ok {
			socketDeployMtxs[args.AppID] = &sync.Mutex{}
		}
		socketDeployMtxs[args.AppID].Lock()
		defer socketDeployMtxs[args.AppID].Unlock()

		log.Println("received a deploy request for", args.AppSlug)

		var applyResult, helmResult *commandResult
		var applyError, helmError error
		defer func() {
			if applyError != nil {
				applyResult = &commandResult{
					hasErr:      true,
					multiStdout: [][]byte{},
					multiStderr: [][]byte{[]byte(applyError.Error())},
				}
			}
			if helmError != nil {
				helmResult = &commandResult{
					hasErr:      true,
					multiStdout: [][]byte{},
					multiStderr: [][]byte{[]byte(helmError.Error())},
				}
			}

			err := c.sendResult(args, nil, applyResult, helmResult)
			if err != nil {
				log.Printf("failed to report result: %v", err)
			}
		}()

		applyResult, applyError = c.deployManifests(args)
		if applyError != nil {
			log.Printf("falied to deploy manifests: %v", applyError)
			return
		}

		helmResult, helmError = c.deployHelmCharts(args)
		if helmError != nil {
			log.Printf("falied to deploy helm charts: %v", helmError)
			return
		}

		c.shutdownNamespacesInformer()
		if len(c.watchedNamespaces) > 0 {
			c.runNamespacesInformer()
		}

	})
	if err != nil {
		return errors.Wrap(err, "failed to add deploy handler")
	}

	err = socketClient.On("appInformers", func(h *socket.Channel, args InformRequest) {
		log.Printf("received an inform event: %#v", args)
		c.applyAppInformers(args.AppID, args.Sequence, args.Informers)
	})
	if err != nil {
		return errors.Wrap(err, "failed to add inform handler")
	}

	return nil
}

func (c *Client) deployManifests(args ApplicationManifests) (*commandResult, error) {
	if args.PreviousManifests != "" {
		if err := c.diffAndRemovePreviousManifests(args); err != nil {
			return nil, errors.Wrapf(err, "failed to remove previous manifests")
		}
	}

	for _, additionalNamespace := range args.AdditionalNamespaces {
		if additionalNamespace == "*" {
			continue
		}

		if err := c.ensureNamespacePresent(additionalNamespace); err != nil {
			// we don't fail here...
			log.Printf("error creating namespace: %s", err.Error())
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
	c.imagePullSecret = args.ImagePullSecret
	c.watchedNamespaces = args.AdditionalNamespaces

	result, err := c.ensureResourcesPresent(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deploy")
	}

	return result, nil
}

func (c *Client) deployHelmCharts(args ApplicationManifests) (*commandResult, error) {
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}

	var prevHelmDir string
	if len(args.PreviousCharts) > 0 {
		tmpDir, err := ioutil.TempDir("", "helm")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create temp dir for previous charts")
		}
		defer os.RemoveAll(tmpDir)

		err = ioutil.WriteFile(path.Join(tmpDir, "archive.tar.gz"), args.PreviousCharts, 0644)
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
	if len(args.Charts) > 0 {
		tmpDir, err := ioutil.TempDir("", "helm")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create temp dir to stage currently deployed archive")
		}
		defer os.RemoveAll(tmpDir)

		err = ioutil.WriteFile(path.Join(tmpDir, "archive.tar.gz"), args.Charts, 0644)
		if err != nil {
			return nil, errors.Wrap(err, "failed to write current archive")
		}

		curHelmDir = path.Join(tmpDir, "currhelm")
		if err := os.MkdirAll(curHelmDir, 0755); err != nil {
			return nil, errors.Wrap(err, "failed to create dir to stage currently deployed archive")
		}

		if err := tarGz.Unarchive(path.Join(tmpDir, "archive.tar.gz"), curHelmDir); err != nil {
			return nil, errors.Wrap(err, "falied to unarchive current helm archive")
		}
	}

	removedCharts, err := getRemovedCharts(prevHelmDir, curHelmDir)
	if err != nil {
		return nil, errors.Wrap(err, "falied to find removed charts")
	}

	if len(removedCharts) > 0 {
		err := c.uninstallWithHelm(prevHelmDir, args.Namespace, removedCharts)
		if err != nil {
			return nil, errors.Wrap(err, "falied to uninstall helm charts")
		}
	}

	if len(args.Charts) > 0 {
		installResult, err = c.installWithHelm(curHelmDir, args.Namespace)
		if err != nil {
			return nil, errors.Wrap(err, "falied to install helm charts")
		}
	}

	return installResult, nil
}

func (c *Client) sendResult(applicationManifests ApplicationManifests, dryRunResult *commandResult, applyResult *commandResult, helmResult *commandResult) error {
	if applicationManifests.ResultCallback == "" {
		return nil
	}

	uri := fmt.Sprintf("%s%s", c.APIEndpoint, applicationManifests.ResultCallback)
	log.Printf("Reporting results to %q", uri)

	result := struct {
		AppID        string `json:"appId"`
		IsError      bool   `json:"isError"`
		DryrunStdout []byte `json:"dryrunStdout"`
		DryrunStderr []byte `json:"dryrunStderr"`
		ApplyStdout  []byte `json:"applyStdout"`
		ApplyStderr  []byte `json:"applyStderr"`
		HelmStdout   []byte `json:"helmStdout"`
		HelmStderr   []byte `json:"helmStderr"`
	}{
		AppID: applicationManifests.AppID,
	}

	isError := false
	if dryRunResult != nil {
		isError = isError || dryRunResult.hasErr
		result.DryrunStdout = bytes.Join(dryRunResult.multiStdout, []byte("\n"))
		result.DryrunStderr = bytes.Join(dryRunResult.multiStderr, []byte("\n"))
	}

	if applyResult != nil {
		isError = isError || applyResult.hasErr
		result.ApplyStdout = bytes.Join(applyResult.multiStdout, []byte("\n"))
		result.ApplyStderr = bytes.Join(applyResult.multiStderr, []byte("\n"))
	}

	if helmResult != nil {
		isError = isError || helmResult.hasErr
		result.HelmStdout = bytes.Join(helmResult.multiStdout, []byte("\n"))
		result.HelmStderr = bytes.Join(helmResult.multiStderr, []byte("\n"))
	}

	result.IsError = isError

	b, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to marshal results")
	}

	req, err := http.NewRequest("PUT", uri, bytes.NewBuffer(b))
	if err != nil {
		return errors.Wrap(err, "could not create result request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("", c.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not execute result PUT request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from kotsadm server: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) applyAppInformers(appID string, sequence int64, informerStrings []types.StatusInformerString) {
	var informers []types.StatusInformer
	for _, str := range informerStrings {
		informer, err := str.Parse()
		if err != nil {
			log.Printf(fmt.Sprintf("failed to parse informer %s: %s", str, err.Error()))
			continue // don't stop
		}
		informers = append(informers, informer)
	}
	if len(informers) > 0 {
		c.appStateMonitor.Apply(appID, sequence, informers)
	}
}

func (c *Client) sendAppStatus(appStatus types.AppStatus) error {
	b, err := json.Marshal(appStatus)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request")
	}

	uri := fmt.Sprintf("%s/api/v1/appstatus", c.APIEndpoint)

	req, err := http.NewRequest("PUT", uri, bytes.NewBuffer(b))
	if err != nil {
		return errors.Wrap(err, "could not create app status request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("", c.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not execute app status PUT request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code from kotsadm server: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) getApplier(kubectlVersion string) (*applier.Kubectl, error) {
	kubectl, err := util.FindKubectlVersion(kubectlVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find kubectl")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get in cluster config")
	}

	return applier.NewKubectl(kubectl, config), nil
}
