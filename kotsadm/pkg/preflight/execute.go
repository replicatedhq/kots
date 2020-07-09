package preflight

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	troubleshootv1beta1 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta1"
	"github.com/replicatedhq/troubleshoot/pkg/preflight"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
)

// execute will execute the preflights using spec in preflightSpec.
// This spec should be rendered, no template functions remaining
func execute(appID string, sequence int64, preflightSpec *troubleshootv1beta1.Preflight, ignorePermissionErrors bool) error {
	logger.Debug("executing preflight checks",
		zap.String("appID", appID),
		zap.Int64("sequence", sequence))

	progressChan := make(chan interface{}, 0) // non-zero buffer will result in missed messages
	defer close(progressChan)

	go func() {
		for {
			msg, ok := <-progressChan
			if ok {
				logger.Debugf("%v", msg)
			} else {
				return
			}
		}
	}()

	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to read in cluster config")
	}

	collectOpts := troubleshootpreflight.CollectOpts{
		Namespace:              "",
		IgnorePermissionErrors: ignorePermissionErrors,
		ProgressChan:           progressChan,
		KubernetesRestConfig:   restConfig,
	}

	logger.Debug("preflight collect phase")
	collectResults, err := troubleshootpreflight.Collect(collectOpts, preflightSpec)
	if err != nil && !isPermissionsError(err) {
		return errors.Wrap(err, "failed to collect")
	}

	uploadPreflightResults := &troubleshootpreflight.UploadPreflightResults{}
	if isPermissionsError(err) {
		logger.Debug("skipping analyze due to RBAC errors")
		rbacErrors := []*troubleshootpreflight.UploadPreflightError{}
		for _, collector := range collectResults.Collectors {
			for _, e := range collector.RBACErrors {
				rbacErrors = append(rbacErrors, &preflight.UploadPreflightError{
					Error: e.Error(),
				})
			}
		}
		uploadPreflightResults.Errors = rbacErrors
	} else {
		logger.Debug("preflight analyze phase")
		analyzeResults := collectResults.Analyze()

		// the typescript api added some flair to this result
		// so let's keep it for compatibility
		// MORE TYPES!
		results := []*troubleshootpreflight.UploadPreflightResult{}
		for _, analyzeResult := range analyzeResults {
			uploadPreflightResult := &troubleshootpreflight.UploadPreflightResult{
				IsFail:  analyzeResult.IsFail,
				IsWarn:  analyzeResult.IsWarn,
				IsPass:  analyzeResult.IsPass,
				Title:   analyzeResult.Title,
				Message: analyzeResult.Message,
				URI:     analyzeResult.URI,
			}

			results = append(results, uploadPreflightResult)
		}
		uploadPreflightResults.Results = results
	}

	logger.Debug("preflight marshalling")
	b, err := json.Marshal(uploadPreflightResults)
	if err != nil {
		return errors.Wrap(err, "failed to marshal results")
	}
	db := persistence.MustGetPGSession()
	query := `update app_downstream_version set preflight_result = $1, preflight_result_created_at = $2,
status = (case when status = 'deployed' then 'deployed' else 'pending' end)
where app_id = $3 and parent_sequence = $4`

	_, err = db.Exec(query, b, time.Now(), appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to write preflight results")
	}

	// deploy first version if preflight checks passed
	preflightState := getPreflightState(uploadPreflightResults)
	if sequence == 0 && preflightState == "pass" {
		err := version.DeployVersion(appID, sequence)
		if err != nil {
			return errors.Wrap(err, "failed to deploy first version")
		}
	}

	return nil
}

func isPermissionsError(err error) bool {
	// TODO: make an error type in troubleshoot for this instead of hardcoding the message
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "insufficient permissions to run all collectors")
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
