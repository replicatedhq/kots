package preflight

import (
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight/types"
	troubleshootanalyze "github.com/replicatedhq/troubleshoot/pkg/analyze"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	troubleshootcollect "github.com/replicatedhq/troubleshoot/pkg/collect"
	"github.com/replicatedhq/troubleshoot/pkg/preflight"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
)

// Execute will Execute the preflights using spec in preflightSpec.
// This spec should be rendered, no template functions remaining
func Execute(preflightSpec *troubleshootv1beta2.Preflight, ignorePermissionErrors bool, setProgress func(progress map[string]interface{}) error, setResults func(results *types.PreflightResults) error) (*types.PreflightResults, error) {
	progressChan := make(chan interface{}, 0) // non-zero buffer will result in missed messages
	defer close(progressChan)

	var preflightRunError error
	completeMx := sync.Mutex{}
	isComplete := false
	go func() {
		for {
			msg, ok := <-progressChan
			if !ok {
				return
			}

			if err, ok := msg.(error); ok {
				logger.Errorf("error while running preflights: %v", err)
			} else {
				switch m := msg.(type) {
				case preflight.CollectProgress:
					logger.Infof("preflight progress: %s", m.String())
				default:
					logger.Infof("preflight progress: %+v", msg)
				}
			}

			collectProgress, ok := msg.(preflight.CollectProgress)
			if !ok {
				continue
			}

			// TODO: We need a nice title to display
			progress := map[string]interface{}{
				"completedCount": collectProgress.CompletedCount,
				"totalCount":     collectProgress.TotalCount,
				"currentName":    collectProgress.CurrentName,
				"currentStatus":  collectProgress.CurrentStatus,
				"updatedAt":      time.Now().Format(time.RFC3339),
			}

			completeMx.Lock()
			if !isComplete {
				if err := setProgress(progress); err != nil {
					logger.Error(errors.Wrap(err, "failed to set preflight progress"))
				}
			}
			completeMx.Unlock()
		}
	}()

	uploadPreflightResults := &types.PreflightResults{}
	defer func() {
		completeMx.Lock()
		defer completeMx.Unlock()

		isComplete = true

		if preflightRunError != nil {
			if uploadPreflightResults.Errors == nil {
				uploadPreflightResults.Errors = []*types.PreflightError{}
			}
			uploadPreflightResults.Errors = append(uploadPreflightResults.Errors, &types.PreflightError{
				Error:  preflightRunError.Error(),
				IsRBAC: false,
			})
		}
		if err := setResults(uploadPreflightResults); err != nil {
			logger.Error(errors.Wrap(err, "failed to set preflight results"))
			return
		}
	}()

	restConfig, err := k8sutil.GetClusterConfig()
	if err != nil {
		preflightRunError = err
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	preflightSpec.Spec.Collectors = troubleshootcollect.DedupCollectors(preflightSpec.Spec.Collectors)
	preflightSpec.Spec.Analyzers = troubleshootanalyze.DedupAnalyzers(preflightSpec.Spec.Analyzers)

	collectOpts := troubleshootpreflight.CollectOpts{
		Namespace:              "",
		IgnorePermissionErrors: ignorePermissionErrors,
		ProgressChan:           progressChan,
		KubernetesRestConfig:   restConfig,
	}

	logger.Info("preflight collect phase")
	collectResults, err := troubleshootpreflight.Collect(collectOpts, preflightSpec)
	if err != nil && !isPermissionsError(err) {
		preflightRunError = err
		return nil, errors.Wrap(err, "failed to collect")
	}

	clusterCollectResult, ok := collectResults.(troubleshootpreflight.ClusterCollectResult)
	if !ok {
		preflightRunError = errors.Errorf("unexpected result type: %T", collectResults)
		return nil, preflightRunError
	}

	if isPermissionsError(err) {
		logger.Debug("skipping analyze due to RBAC errors")
		rbacErrors := []*types.PreflightError{}
		for _, collector := range clusterCollectResult.Collectors {
			for _, e := range collector.GetRBACErrors() {
				rbacErrors = append(rbacErrors, &types.PreflightError{
					Error:  e.Error(),
					IsRBAC: true,
				})
			}
		}
		uploadPreflightResults.Errors = rbacErrors
	} else {
		logger.Info("preflight analyze phase")
		analyzeResults := collectResults.Analyze()

		// the typescript api added some flair to this result
		// so let's keep it for compatibility
		// MORE TYPES!
		results := []*troubleshootpreflight.UploadPreflightResult{}
		for _, analyzeResult := range analyzeResults {
			uploadPreflightResult := &troubleshootpreflight.UploadPreflightResult{
				Strict:  analyzeResult.Strict,
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

	return uploadPreflightResults, nil
}

func isPermissionsError(err error) bool {
	// TODO: make an error type in troubleshoot for this instead of hardcoding the message
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "insufficient permissions to run all collectors")
}
