package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/operator/pkg/applier"
	"github.com/replicatedhq/kotsadm/operator/pkg/socket"
	"github.com/replicatedhq/kotsadm/operator/pkg/socket/transport"
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

type Client struct {
	APIEndpoint string
	Token       string
}

func (c *Client) Run() error {
	fmt.Println("Starting kotsadm-operator loop")

	for {
		err := c.connect()
		if err != nil {
			// this needs a backoff
			fmt.Printf("unable to connect to api\n")
			time.Sleep(time.Second * 2)
			continue
		}

		// some easy backoff for now
		time.Sleep(time.Second * 2)
	}
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

	fmt.Println("connecting to api")
	socketClient, err := socket.Dial(socket.GetUrl(u.Hostname(), port, c.Token, false), transport.GetDefaultWebsocketTransport())
	if err != nil {
		return errors.Wrap(err, "failed to connect")
	}

	defer socketClient.Close()

	err = socketClient.On("preflight", func(h *socket.Channel, args PreflightRequest) {
		fmt.Printf("received a preflight event: %#v\n", args)
		if err := runPreflight(args.URI); err != nil {
			fmt.Printf("error running preflight: %s\n", err.Error())
		}
	})
	if err != nil {
		return errors.Wrap(err, "error in prefight handler")
	}

	err = socketClient.On("deploy", func(h *socket.Channel, args ApplicationManifests) {
		fmt.Printf("received a deploy request\n")
		if err := c.ensureResourcesPresent(args); err != nil {
			fmt.Printf("error deploying: %s\n", err.Error())
		}
	})
	if err != nil {
		return errors.Wrap(err, "error in deploy handler")
	}

	err = socketClient.On(socket.OnConnection, func(h *socket.Channel) {
		hasConnected = true
	})
	if err != nil {
		return errors.Wrap(err, "error in connected handler")
	}

	err = socketClient.On(socket.OnDisconnection, func(h *socket.Channel) {
		isUnexpectedlyDisconnected = true
	})
	if err != nil {
		return errors.Wrap(err, "error in disconnected handler")
	}

	// wait for a connection for at least 2 seconds
	time.Sleep(time.Second * 2)
	if !hasConnected {
		return nil // allow another attempt
	}

	for {
		if isUnexpectedlyDisconnected {
			return nil
		}

		time.Sleep(time.Second * 2)
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

	client := &http.Client{}

	uri := fmt.Sprintf("%s/api/v1/deploy/result", c.APIEndpoint)

	fmt.Printf("Reporting results to %q\n", uri)
	req, err := http.NewRequest("PUT", uri, bytes.NewBuffer(b))
	req.Header["Content-Type"] = []string{"application/json"}
	req.SetBasicAuth("", c.Token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from kotsadm server: %d", resp.StatusCode)
	}

	return nil
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

	fmt.Println("dry run applying manifests(s)")
	drrunStdout, dryrunStderr, dryRunErr := kubernetesApplier.Apply(applicationManifests.Namespace, decoded, true)
	if dryRunErr != nil {
		fmt.Printf("stdout (dryrun) = %s\n", drrunStdout)
		fmt.Printf("stderr (dryrun) = %s\n", dryrunStderr)
	}

	var applyStdout []byte
	var applyStderr []byte
	var applyErr error
	if dryRunErr == nil {
		fmt.Println("applying manifest(s)")
		stdout, stderr, err := kubernetesApplier.Apply(applicationManifests.Namespace, decoded, false)
		if err != nil {
			fmt.Printf("stdout (apply) = %s\n", stderr)
			fmt.Printf("stderr (apply) = %s\n", stderr)
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
