package plan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	plantypes "github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
)

func UpdateStepStatus(params types.UpgradeServiceParams, status plantypes.PlanStepStatus, description string, output string) error {
	body, err := json.Marshal(map[string]string{
		"versionLabel":      params.UpdateVersionLabel,
		"status":            string(status),
		"statusDescription": description,
		"output":            output,
	})
	if err != nil {
		return errors.Wrap(err, "marshal request body")
	}

	url := fmt.Sprintf("http://localhost:3000/api/v1/app/%s/plan/%s", params.AppSlug, params.PlanStepID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return errors.Wrap(err, "create request")
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("send request, status code: %d", resp.StatusCode)
	}

	return nil
}
