package preflight

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
	troubleshootv1beta1 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta1"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
)

// execute will execute the preflights using spec in preflightSpec.
// This spec should be rendered, no template functions remaining
func execute(appID string, sequence int64, preflightSpec *troubleshootv1beta1.Preflight) error {
	logger.Debug("executing preflight checks",
		zap.String("appID", appID),
		zap.Int64("sequence", sequence))

	progressChan := make(chan interface{}, 0) // non-zero buffer will result in missed messages

	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to read in cluster config")
	}

	collectOpts := troubleshootpreflight.CollectOpts{
		Namespace:              "",
		IgnorePermissionErrors: false,
		ProgressChan:           progressChan,
		KubernetesRestConfig:   restConfig,
	}

	logger.Debug("preflight collect phase")
	collectResults, err := troubleshootpreflight.Collect(collectOpts, preflightSpec)
	if err != nil {
		return errors.Wrap(err, "failed to collect")
	}

	logger.Debug("preflight analyze phase")
	analyzeResults := collectResults.Analyze()
	if err != nil {
		return errors.Wrap(err, "failed to analyze")
	}

	// the typescript api added some flair to this result
	// so let's keep it for compatibility
	// MORE TYPES!
	uploadPreflightResults := &troubleshootpreflight.UploadPreflightResults{
		Results: []*troubleshootpreflight.UploadPreflightResult{},
	}
	for _, analyzeResult := range analyzeResults {
		uploadPreflightResult := &troubleshootpreflight.UploadPreflightResult{
			IsFail:  analyzeResult.IsFail,
			IsWarn:  analyzeResult.IsWarn,
			IsPass:  analyzeResult.IsPass,
			Title:   analyzeResult.Title,
			Message: analyzeResult.Message,
			URI:     analyzeResult.URI,
		}

		uploadPreflightResults.Results = append(uploadPreflightResults.Results, uploadPreflightResult)
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

	return nil
}
