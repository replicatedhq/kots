package preflight

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/preflight"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
)

// execute will execute the preflights using spec in preflightSpec.
// This spec should be rendered, no template functions remaining
func execute(appID string, sequence int64, preflightSpec *troubleshootv1beta2.Preflight, ignorePermissionErrors bool) (*troubleshootpreflight.UploadPreflightResults, error) {
	logger.Debug("executing preflight checks",
		zap.String("appID", appID),
		zap.Int64("sequence", sequence))

	progressChan := make(chan interface{}, 0) // non-zero buffer will result in missed messages
	defer close(progressChan)

	completeMx := sync.Mutex{}
	isComplete := false
	go func() {
		for {
			msg, ok := <-progressChan
			if !ok {
				return
			}

			logger.Debugf("%v", msg)

			progress, ok := msg.(preflight.CollectProgress)
			if !ok {
				continue
			}

			// TODO: We need a nice title to display
			progresBytes, err := json.Marshal(map[string]interface{}{
				"completedCount": progress.CompletedCount,
				"totalCount":     progress.TotalCount,
				"currentName":    progress.CurrentName,
				"currentStatus":  progress.CurrentStatus,
				"updatedAt":      time.Now().Format(time.RFC3339),
			})
			if err != nil {
				continue
			}

			completeMx.Lock()
			if !isComplete {
				_ = store.GetStore().SetPreflightProgress(appID, sequence, string(progresBytes))
			}
			completeMx.Unlock()
		}
	}()

	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read in cluster config")
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
		return nil, errors.Wrap(err, "failed to collect")
	}

	clusterCollectResult, ok := collectResults.(troubleshootpreflight.ClusterCollectResult)
	if !ok {
		return nil, errors.Errorf("unexpected result type: %T", collectResults)
	}

	uploadPreflightResults := &troubleshootpreflight.UploadPreflightResults{}
	if isPermissionsError(err) {
		logger.Debug("skipping analyze due to RBAC errors")
		rbacErrors := []*troubleshootpreflight.UploadPreflightError{}
		for _, collector := range clusterCollectResult.Collectors {
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
		return uploadPreflightResults, errors.Wrap(err, "failed to marshal results")
	}

	completeMx.Lock()
	defer completeMx.Unlock()

	isComplete = true
	if err := store.GetStore().SetPreflightResults(appID, sequence, b); err != nil {
		return uploadPreflightResults, errors.Wrap(err, "failed to set preflight results")
	}

	return uploadPreflightResults, nil
}

func isPermissionsError(err error) bool {
	// TODO: make an error type in troubleshoot for this instead of hardcoding the message
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "insufficient permissions to run all collectors")
}
