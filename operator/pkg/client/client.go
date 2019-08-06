package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/operator/pkg/helm"
)

type Client struct {
	APIEndpoint string
	Token       string
}

func (c *Client) Run() error {
	for {
		_, err := getDesiredStateFromKotsadmServer(c.APIEndpoint, c.Token)
		if err != nil {
			return errors.Wrap(err, "failed to get destired state from server")
		}

		time.Sleep(time.Second * 10)
	}
}

type DesiredState struct {
	Present []string `json:"present"`
	Missing []string `json:"missing"`
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
		return nil, fmt.Errorf("unexpected status code from kotsasdm server: %d", resp.StatusCode)
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

func reporCurrentStateToKotsadmServer(apiEndpoint string, token string, helmApplications []*helm.HelmApplication) error {
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
		return fmt.Errorf("unexpected status code from kotadm server: %d", resp.StatusCode)
	}

	return nil
}
