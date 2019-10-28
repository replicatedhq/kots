package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/operator/pkg/applier"
	"github.com/replicatedhq/kotsadm/operator/pkg/appstate"
	"github.com/replicatedhq/kotsadm/operator/pkg/appstate/types"
	"github.com/replicatedhq/kotsadm/operator/pkg/socket"
	"github.com/replicatedhq/kotsadm/operator/pkg/socket/transport"
	"github.com/replicatedhq/kotsadm/operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

var (
	PollInterval = time.Second * 10
)

type ApplicationManifests struct {
	AppID     string `json:"app_id"`
	Namespace string `json:"namespace"`
	Manifests string `json:"manifests"`
}

// DesiredState is what we receive from the kotsadm-api server
type DesiredState struct {
	Present   []ApplicationManifests `json:"present"`
	Missing   map[string][]string    `json:"missing"`
	Preflight []string               `json:"preflight"`
}

type PreflightRequest struct {
	URI string `json:"uri"`
}

type SupportBundleRequest struct {
	URI string `json:"uri"`
}

type InformRequest struct {
	Informers []types.StatusInformer `json:"informers"`
}

type Client struct {
	APIEndpoint string
	Token       string

	appStateMonitor *appstate.Monitor
}

func (c *Client) Run() error {
	log.Println("Starting kotsadm-operator loop")

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
	throttled := util.NewThrottle(time.Second)

	for appStatus := range c.appStateMonitor.AppStatusChan() {
		throttled(func() {
			log.Printf("Sending app status %#v", appStatus)
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
	socketClient, err := socket.Dial(socket.GetUrl(u.Hostname(), port, c.Token, false), transport.GetDefaultWebsocketTransport())
	if err != nil {
		return errors.Wrap(err, "failed to connect")
	}

	defer socketClient.Close()

	targetNamespace := os.Getenv("DEFAULT_NAMESPACE")
	if targetNamespace == "" {
		targetNamespace = corev1.NamespaceDefault
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get in cluster config")
	}

	c.appStateMonitor, err = appstate.NewMonitor(config, targetNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to create appstate monitor")
	}
	defer c.appStateMonitor.Shutdown()

	go c.runAppStateMonitor()

	err = socketClient.On("preflight", func(h *socket.Channel, args PreflightRequest) {
		log.Printf("received a preflight event: %#v", args)
		if err := runPreflight(args.URI); err != nil {
			log.Printf("error running preflight: %s", err.Error())
		}
	})
	if err != nil {
		return errors.Wrap(err, "error in prefight handler")
	}

	err = socketClient.On("deploy", func(h *socket.Channel, args ApplicationManifests) {
		log.Println("received a deploy request")
		if err := c.ensureResourcesPresent(args); err != nil {
			log.Printf("error deploying: %s", err.Error())
		}
	})
	if err != nil {
		return errors.Wrap(err, "error in deploy handler")
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
		return errors.Wrap(err, "error in support bundle handler")
	}

	err = socketClient.On("inform", func(h *socket.Channel, args InformRequest) {
		log.Printf("received an inform event: %#v", args)
		if err := c.applyInformers(args.Informers); err != nil {
			log.Printf("error running informer: %s", err.Error())
		}
	})
	if err != nil {
		return errors.Wrap(err, "error in inform handler")
	}

	err = socketClient.On(socket.OnConnection, func(h *socket.Channel) {
		hasConnected = true
	})
	if err != nil {
		return errors.Wrap(err, "error in connected handler")
	}

	err = socketClient.On(socket.OnDisconnection, func(h *socket.Channel, args interface{}) {
		log.Printf("disconnected %#v", args)
		isUnexpectedlyDisconnected = true
	})
	if err != nil {
		return errors.Wrap(err, "error in disconnected handler")
	}

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

func (c *Client) sendResult(applicationManifests ApplicationManifests, isError bool, dryrunStdout []byte, dryrunStderr []byte, applyStdout []byte, applyStderr []byte) error {
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

	uri := fmt.Sprintf("%s/api/v1/deploy/result", c.APIEndpoint)

	log.Printf("Reporting results to %q", uri)
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

func runPreflight(preflightURI string) error {
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

	return kubernetesApplier.Preflight(preflightURI)
}

func (c *Client) applyInformers(informers []types.StatusInformer) error {
	c.appStateMonitor.Apply(informers)
	return nil
}

func (c *Client) sendAppStatus(appStatus types.AppStatus) error {
	appStatusRequest := struct {
		AppStatus types.AppStatus `json:"app_status"`
	}{
		AppStatus: appStatus,
	}

	b, err := json.Marshal(appStatusRequest)
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

func (c *Client) ensureResourcesPresent(applicationManifests ApplicationManifests) error {
	decoded, err := base64.StdEncoding.DecodeString(applicationManifests.Manifests)
	if err != nil {
		return errors.Wrap(err, "failed to decode manifests")
	}

	// TODO sort, order matters
	// TODO should we split multi-doc to retry on failed?

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

	// this is pretty raw, and required kubectl...  we should
	// consider some other options here?
	kubernetesApplier := applier.NewKubectl(kubectl, preflight, supportBundle, config)

	log.Println("dry run applying manifests(s)")
	drrunStdout, dryrunStderr, dryRunErr := kubernetesApplier.Apply(applicationManifests.Namespace, decoded, true)
	if dryRunErr != nil {
		log.Printf("stdout (dryrun) = %s", drrunStdout)
		log.Printf("stderr (dryrun) = %s", dryrunStderr)
	}

	var applyStdout []byte
	var applyStderr []byte
	var applyErr error
	if dryRunErr == nil {
		log.Println("applying manifest(s)")
		stdout, stderr, err := kubernetesApplier.Apply(applicationManifests.Namespace, decoded, false)
		if err != nil {
			log.Printf("stdout (apply) = %s", stderr)
			log.Printf("stderr (apply) = %s", stderr)
		}

		applyStdout = stdout
		applyStderr = stderr
		applyErr = err
	}

	hasErr := applyErr != nil || dryRunErr != nil
	if err := c.sendResult(applicationManifests, hasErr, drrunStdout, dryrunStderr, applyStdout, applyStderr); err != nil {
		return errors.Wrap(err, "failed to report status")
	}

	return nil
}
