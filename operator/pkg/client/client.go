package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/operator/pkg/applier"
	"github.com/replicatedhq/kotsadm/operator/pkg/appstate"
	"github.com/replicatedhq/kotsadm/operator/pkg/appstate/types"
	"github.com/replicatedhq/kotsadm/operator/pkg/socket"
	"github.com/replicatedhq/kotsadm/operator/pkg/socket/transport"
	"github.com/replicatedhq/kotsadm/operator/pkg/util"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	PollInterval = time.Second * 10
)

type ApplicationManifests struct {
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
}

// DesiredState is what we receive from the kotsadm-api server
type DesiredState struct {
	Present   []ApplicationManifests `json:"present"`
	Missing   map[string][]string    `json:"missing"`
	Preflight []string               `json:"preflight"`
}

type PreflightRequest struct {
	URI               string `json:"uri"`
	IgnorePermissions bool   `json:"ignorePermissions"`
}

type SupportBundleRequest struct {
	URI string `json:"uri"`
}

type InformRequest struct {
	AppID     string                       `json:"app_id"`
	Informers []types.StatusInformerString `json:"informers"`
}

type Client struct {
	APIEndpoint     string
	Token           string
	TargetNamespace string

	watchedNamespaces []string
	imagePullSecret   string

	appStateMonitor   *appstate.Monitor
	hookStopChans     []chan struct{}
	namespaceStopChan chan struct{}
}

// Run is the main entrypoint of the operator when running in standard, normal operations
func (c *Client) Run() error {
	log.Println("Starting kotsadm-operator loop")

	c.runHooksInformer()
	defer c.shutdownHooksInformer()

	defer c.shutdownNamespacesInformer()

	for {
		err := c.connect()
		if err != nil {
			// this needs a backoff
			log.Println("unable to connect to api")
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

	log.Println("connecting to api")
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

	err = socketClient.On("preflight", func(h *socket.Channel, args PreflightRequest) {
		log.Printf("received a preflight event: %#v", args)
		if err := runPreflight(args.URI, args.IgnorePermissions); err != nil {
			log.Printf("error running preflight: %s", err.Error())
		}
	})
	if err != nil {
		return errors.Wrap(err, "failed in preflight handler")
	}

	err = socketClient.On("deploy", func(h *socket.Channel, args ApplicationManifests) {
		log.Println("received a deploy request")

		if args.PreviousManifests != "" {
			if err := c.diffAndRemovePreviousManifests(args); err != nil {
				log.Printf("error diffing and removing previous manifests: %s", err.Error())
				return
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
		}
		c.imagePullSecret = args.ImagePullSecret
		c.watchedNamespaces = args.AdditionalNamespaces

		if err := c.ensureResourcesPresent(args); err != nil {
			log.Printf("error deploying: %s", err.Error())
		}

		c.shutdownNamespacesInformer()
		c.runNamespacesInformer()

	})
	if err != nil {
		return errors.Wrap(err, "failed in deploy handler")
	}

	err = socketClient.On("supportbundle", func(h *socket.Channel, args SupportBundleRequest) {
		log.Println("received a support bundle request")
		go func() {
			// This is in a goroutine because if we disconnect and reconnect to the
			// websocket, we will want to report that it's completed...
			err := runSupportBundle(args.URI)
			log.Printf("support bundle run completed with err = %#v", err)
			if err != nil {
				log.Printf("error running support bundle: %s", err.Error())
			}
		}()
	})
	if err != nil {
		return errors.Wrap(err, "failed in support bundle handler")
	}

	err = socketClient.On("appInformers", func(h *socket.Channel, args InformRequest) {
		log.Printf("received an inform event: %#v", args)
		if err := c.applyAppInformers(args.AppID, args.Informers); err != nil {
			log.Printf("error running informer: %s", err.Error())
		}
	})
	if err != nil {
		return errors.Wrap(err, "failed in inform handler")
	}

	return nil
}

func (c *Client) sendResult(applicationManifests ApplicationManifests, isError bool, dryrunStdout []byte, dryrunStderr []byte, applyStdout []byte, applyStderr []byte) error {
	if applicationManifests.ResultCallback == "" {
		return nil
	}

	uri := fmt.Sprintf("%s%s", c.APIEndpoint, applicationManifests.ResultCallback)
	log.Printf("Reporting results to %q", uri)

	applyResult := struct {
		AppID        string `json:"app_id"`
		IsError      bool   `json:"is_error"`
		DryrunStdout []byte `json:"dryrun_stdout"`
		DryrunStderr []byte `json:"dryrun_stderr"`
		ApplyStdout  []byte `json:"apply_stdout"`
		ApplyStderr  []byte `json:"apply_stderr"`
	}{
		applicationManifests.AppID,
		isError,
		dryrunStdout,
		dryrunStderr,
		applyStdout,
		applyStderr,
	}

	b, err := json.Marshal(applyResult)
	if err != nil {
		return errors.Wrap(err, "failed to marshal results")
	}

	req, err := http.NewRequest("PUT", uri, bytes.NewBuffer(b))
	req.Header["Content-Type"] = []string{"application/json"}
	req.SetBasicAuth("", c.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from kotsadm server: %d", resp.StatusCode)
	}

	return nil
}

func runSupportBundle(collectorURI string) error {
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return errors.Wrap(err, "failed to find kubectl")
	}

	preflight := ""
	localPreflight, err := exec.LookPath("support-bundle")
	if err == nil {
		preflight = localPreflight
	}

	supportBundle := ""
	localSupportBundle, err := exec.LookPath("support-bundle")
	if err == nil {
		supportBundle = localSupportBundle
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get in cluster config")
	}

	kubernetesApplier := applier.NewKubectl(kubectl, preflight, supportBundle, config)

	return kubernetesApplier.SupportBundle(collectorURI)
}

func runPreflight(preflightURI string, ignorePermissions bool) error {
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return errors.Wrap(err, "failed to find kubectl")
	}

	preflight := ""
	localPreflight, err := exec.LookPath("preflight")
	if err == nil {
		preflight = localPreflight
	}

	supportBundle := ""
	localSupportBundle, err := exec.LookPath("support-bundle")
	if err == nil {
		supportBundle = localSupportBundle
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get in cluster config")
	}

	kubernetesApplier := applier.NewKubectl(kubectl, preflight, supportBundle, config)

	return kubernetesApplier.Preflight(preflightURI, ignorePermissions)
}

func (c *Client) applyAppInformers(appID string, informerStrings []types.StatusInformerString) error {
	var informers []types.StatusInformer
	for _, str := range informerStrings {
		informer, err := str.Parse()
		if err != nil {
			return errors.Wrapf(err, "failed to parse informer %s", str)
		}
		informers = append(informers, informer)
	}
	c.appStateMonitor.Apply(appID, informers)
	return nil
}

func (c *Client) sendAppStatus(appStatus types.AppStatus) error {
	b, err := json.Marshal(appStatus)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request")
	}

	uri := fmt.Sprintf("%s/api/v1/appstatus", c.APIEndpoint)

	req, err := http.NewRequest("PUT", uri, bytes.NewBuffer(b))
	req.Header["Content-Type"] = []string{"application/json"}
	req.SetBasicAuth("", c.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
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

	preflight := ""
	localPreflight, err := exec.LookPath("preflight")
	if err == nil {
		preflight = localPreflight
	}

	supportBundle := ""
	localSupportBundle, err := exec.LookPath("support-bundle")
	if err == nil {
		supportBundle = localSupportBundle
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get in cluster config")
	}

	return applier.NewKubectl(kubectl, preflight, supportBundle, config), nil
}
