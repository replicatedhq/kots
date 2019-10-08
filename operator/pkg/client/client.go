package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/operator/pkg/applier"
	"github.com/replicatedhq/kotsadm/operator/pkg/helm"
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

type Client struct {
	APIEndpoint string
	Token       string
}

func (c *Client) Run() error {
	fmt.Println("Starting kotsadm-operator loop")

	for {
		desiredState, err := getDesiredStateFromKotsadmServer(c.APIEndpoint, c.Token)
		if err != nil {
			fmt.Printf("failed to get destired state, waiting to try again: %s\n", err.Error())
			time.Sleep(time.Minute)
			continue
		}

		// Preflight
		for _, preflightURI := range desiredState.Preflight {
			if err := runPreflight(preflightURI); err != nil {
				fmt.Printf("error running preflight: %s\n", err.Error())
			}
		}

		// Deploy
		for _, present := range desiredState.Present {
			if err := c.ensureResourcesPresent(present); err != nil {
				fmt.Printf("error in kubectl apply: %#v\n", err)
				continue
			}
		}

		time.Sleep(PollInterval)
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

func getDesiredStateFromKotsadmServer(apiEndpoint string, token string) (*DesiredState, error) {
	client := &http.Client{}

	uri := fmt.Sprintf("%s/api/v1/deploy/desired", apiEndpoint)

	fmt.Printf("Requesting desired state from %q\n", uri)
	req, err := http.NewRequest("GET", uri, nil)
	req.SetBasicAuth("", token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from kotsadm server: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var desiredState DesiredState
	if err := json.Unmarshal(body, &desiredState); err != nil {
		return nil, err
	}

	return &desiredState, nil
}

func reportCurrentStateToKotsadmServer(apiEndpoint string, token string, helmApplications []*helm.HelmApplication) error {
	currentState := struct {
		HelmApplications []*helm.HelmApplication `json:"helmApplications"`
	}{
		helmApplications,
	}

	b, err := json.Marshal(currentState)
	if err != nil {
		return err
	}

	client := &http.Client{}

	uri := fmt.Sprintf("%s/api/v1/current", apiEndpoint)

	fmt.Printf("Reporting helm charts to %q\n", uri)
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(b))
	req.Header["Content-Type"] = []string{"application/json"}
	req.SetBasicAuth("", token)
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
