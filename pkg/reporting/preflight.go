package reporting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	appstatustypes "github.com/replicatedhq/kots/pkg/api/appstatus/types"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
)

func SendPreflightsReportToReplicatedApp(license *kotsv1beta1.License, appID string, clusterID string, sequence int64, skipPreflights bool, installStatus string, isCLI bool, preflightStatus string, appStatus string) error {
	endpoint := license.Spec.Endpoint
	if !canReport(endpoint) {
		return nil
	}

	urlValues := url.Values{}
	urlValues.Set("sequence", fmt.Sprintf("%d", sequence))
	urlValues.Set("skipPreflights", fmt.Sprintf("%t", skipPreflights))
	urlValues.Set("installStatus", installStatus)
	urlValues.Set("isCLI", fmt.Sprintf("%t", isCLI))
	urlValues.Set("preflightStatus", preflightStatus)
	urlValues.Set("appStatus", appStatus)
	urlValues.Set("kotsVersion", buildversion.Version())

	url := fmt.Sprintf("%s/kots_metrics/preflights/%s/%s?%s", endpoint, appID, clusterID, urlValues.Encode())
	var buf bytes.Buffer
	postReq, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return errors.Wrap(err, "failed to call newrequest")
	}
	postReq.Header.Add("Authorization", license.Spec.LicenseID)
	postReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(postReq)
	if err != nil {
		return errors.Wrap(err, "failed to send preflights reports")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return errors.Errorf("Unexpected status code %d", resp.StatusCode)
	}
	return nil
}

func SendPreflightInfo(appID string, sequence int64, isSkipPreflights bool, isCLI bool) error {
	app, err := store.GetStore().GetApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	if app.IsAirgap {
		logger.Debug("no reporting for airgapped app")
		return nil
	}

	license, err := store.GetStore().GetLatestLicenseForApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to find license for app")
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to list downstreams for app")
	} else if len(downstreams) == 0 {
		err = errors.New("no downstreams for app")
		return err
	}

	clusterID := downstreams[0].ClusterID

	go func() {
		appStatus := appstatustypes.StateMissing
		for start := time.Now(); time.Since(start) < 20*time.Minute; {
			s, err := store.GetStore().GetAppStatus(appID)
			if err != nil {
				logger.Debugf("failed to get app status: %v", err.Error())
				return
			}

			currentDeployedSequence, err := store.GetStore().GetCurrentParentSequence(appID, clusterID)
			if err != nil {
				logger.Debugf("failed to get downstream parent sequence: %v", err.Error())
				return
			}

			if currentDeployedSequence != sequence {
				logger.Debug("deployed sequence has changed")
				return
			}

			if s.Sequence == sequence && s.State == appstatustypes.StateReady {
				appStatus = s.State
				break
			}
			time.Sleep(time.Second * 10)
		}

		preflightState := ""
		var preflightResults *troubleshootpreflight.UploadPreflightResults
		for start := time.Now(); time.Since(start) < 5*time.Minute; {
			p, err := store.GetStore().GetPreflightResults(appID, sequence)
			if err != nil {
				logger.Debugf("failed to get preflight results: %v", err.Error())
				return
			}

			if p.Result != "" {
				if err := json.Unmarshal([]byte(p.Result), &preflightResults); err != nil {
					logger.Debugf("failed to unmarshal preflight results: %v", err.Error())
					return
				}
				preflightState = getPreflightState(preflightResults)
				break
			}
			time.Sleep(time.Second * 10)
		}

		currentVersionStatus, err := store.GetStore().GetStatusForVersion(appID, clusterID, sequence)
		if err != nil {
			logger.Debugf("failed to get status for version: %v", err)
			return
		}

		if err := SendPreflightsReportToReplicatedApp(license, appID, clusterID, sequence, isSkipPreflights, currentVersionStatus, isCLI, preflightState, string(appStatus)); err != nil {
			logger.Debugf("failed to send preflights data to replicated app: %v", err)
			return
		}
	}()
	return nil
}

func getPreflightState(preflightResults *troubleshootpreflight.UploadPreflightResults) string {
	if len(preflightResults.Errors) > 0 {
		return "fail"
	}
	if len(preflightResults.Results) == 0 {
		return "pass"
	}
	state := "pass"
	for _, result := range preflightResults.Results {
		if result.IsFail {
			return "fail"
		} else if result.IsWarn {
			state = "warn"
		}
	}
	return state
}
