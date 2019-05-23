package cluster

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
