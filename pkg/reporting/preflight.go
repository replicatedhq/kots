package reporting

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
)

func WaitAndReportPreflightChecks(appID string, sequence int64, isSkipPreflights bool, isCLI bool) error {
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
		appStatus := appstatetypes.StateMissing
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

			// if user deploy another version and previous version is still running
			if currentDeployedSequence != sequence {
				logger.Debug("deployed sequence has changed")
				return
			}

			if s.Sequence == sequence && s.State == appstatetypes.StateReady {
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

		if err := GetReporter().SubmitPreflightData(license, appID, clusterID, sequence, isSkipPreflights, currentVersionStatus, isCLI, preflightState, string(appStatus)); err != nil {
			logger.Debugf("failed to submit preflight data: %v", err)
			return
		}
	}()
	return nil
}

func getPreflightState(preflightResults *troubleshootpreflight.UploadPreflightResults) string {
	if preflightResults == nil {
		return ""
	}
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
