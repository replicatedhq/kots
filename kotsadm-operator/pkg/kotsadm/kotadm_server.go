package kotsadmserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/replicatedhq/ship-cluster/kotsadm-operator/pkg/helm"
)

type ShipDesiredState struct {
	Present []string `json:"present"`
	Missing []string `json:"missing"`
}

func getDesiredStateFromShipServer(apiEndpoint string, token string) (*ShipDesiredState, error) {
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
		return nil, fmt.Errorf("unexpected status code from ship server: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var shipDesiredState ShipDesiredState
	if err := json.Unmarshal(body, &shipDesiredState); err != nil {
		return nil, err
	}

	return &shipDesiredState, nil
}

func reporCurrentStateToShipServer(apiEndpoint string, token string, helmApplications []*helm.HelmApplication) error {
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
		return fmt.Errorf("unexpected status code from ship server: %d", resp.StatusCode)
	}

	return nil
}
