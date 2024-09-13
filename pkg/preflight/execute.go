package preflight

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight/types"
	troubleshootanalyze "github.com/replicatedhq/troubleshoot/pkg/analyze"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/collect"
	troubleshootcollect "github.com/replicatedhq/troubleshoot/pkg/collect"
	"github.com/replicatedhq/troubleshoot/pkg/constants"
	"github.com/replicatedhq/troubleshoot/pkg/convert"
	"github.com/replicatedhq/troubleshoot/pkg/preflight"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
	troubleshootversion "github.com/replicatedhq/troubleshoot/pkg/version"
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

	// Store collected data in a temp directory
	bundlePath := filepath.Join(os.TempDir(), "last-preflight-result")

	// Clean up the directory if it already exists. If the path does not exist nothing will happen.
	_ = os.RemoveAll(bundlePath)
	err = os.MkdirAll(bundlePath, 0755)
	if err != nil {
		logger.Warnf("failed to create preflight results directory. Proceed without storing the bundle to /tmp dir: %v", err)
		bundlePath = "" // if we can't write to /tmp, don't try to store the bundle
	}

	collectOpts := troubleshootpreflight.CollectOpts{
		Namespace:              "",
		IgnorePermissionErrors: ignorePermissionErrors,
		ProgressChan:           progressChan,
		KubernetesRestConfig:   restConfig,
		BundlePath:             bundlePath,
	}

	logger.Info("preflight collect phase")
	collectResults, err := troubleshootpreflight.Collect(collectOpts, preflightSpec)
	if err != nil && !isPermissionsError(err) {
		preflightRunError = err
		return nil, errors.Wrap(err, "failed to collect")
	}
	isRBACErr := isPermissionsError(err)

	clusterCollectResult, ok := collectResults.(troubleshootpreflight.ClusterCollectResult)
	if !ok {
		preflightRunError = errors.Errorf("unexpected result type: %T", collectResults)
		return nil, preflightRunError
	}

	collectorResults := collect.CollectorResult(clusterCollectResult.AllCollectedData)

	if isRBACErr {
		logger.Warnf("skipping analyze due to RBAC errors")
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
		var analyzeResults []*troubleshootanalyze.AnalyzeResult
		if bundlePath == "" {
			analyzeResults = collectResults.Analyze()
		} else {
			// Its not a bundle if there is no version file in the root directory
			err = saveTSVersionToBundle(collectorResults, bundlePath)
			if err != nil {
				return nil, errors.Wrap(err, "failed to save version file to preflight bundle")
			}
			ctx := context.Background()
			analyzeResults, err = troubleshootanalyze.AnalyzeLocal(ctx, bundlePath, preflightSpec.Spec.Analyzers, nil)
			if err != nil {
				return nil, errors.Wrap(err, "failed to analyze preflights from local bundle")
			}
		}

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
		err = saveAnalysisResultsToBundle(collectorResults, analyzeResults, bundlePath)
		if err != nil {
			logger.Warnf("Ignore storing preflight analysis file to preflight bundle: %v", err)
		}
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

func saveAnalysisResultsToBundle(
	results collect.CollectorResult, analyzeResults []*troubleshootanalyze.AnalyzeResult, bundlePath string,
) error {
	if results == nil {
		return nil
	}

	data := convert.FromAnalyzerResult(analyzeResults)
	analysis, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal analysis")
	}

	err = results.SaveResult(bundlePath, constants.ANALYSIS_FILENAME, bytes.NewBuffer(analysis))
	if err != nil {
		return errors.Wrap(err, "failed to save analysis")
	}

	return nil
}

func saveTSVersionToBundle(results collect.CollectorResult, bundlePath string) error {
	if results == nil {
		return nil
	}

	version, err := troubleshootversion.GetVersionFile()
	if err != nil {
		return errors.Wrap(err, "failed to get version file")
	}

	err = results.SaveResult(bundlePath, constants.VERSION_FILENAME, bytes.NewBuffer([]byte(version)))
	if err != nil {
		return errors.Wrap(err, "failed to save version file")
	}

	return nil
}
