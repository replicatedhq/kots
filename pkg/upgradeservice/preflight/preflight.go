package preflight

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	preflightpkg "github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/preflight/types"
	"github.com/replicatedhq/kots/pkg/registry"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	upgradeservicetypes "github.com/replicatedhq/kots/pkg/upgradeservice/types"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
	"go.uber.org/zap"
)

type PreflightData struct {
	Progress string                 `json:"progress,omitempty"`
	Result   *types.PreflightResult `json:"result"`
}

var PreflightDataFile string

func Init() error {
	tmpDir, err := os.MkdirTemp("", "preflights")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	PreflightDataFile = filepath.Join(tmpDir, "preflights.json")
	return nil
}

func Run(params upgradeservicetypes.UpgradeServiceParams) error {
	kotsKinds, err := kotsutil.LoadKotsKinds(params.AppArchive)
	if err != nil {
		return errors.Wrap(err, "failed to load rendered kots kinds")
	}
	if !kotsKinds.HasPreflights() {
		logger.Info("no preflights found, not running preflights")
		return nil
	}

	registrySettings := registrytypes.RegistrySettings{
		Hostname:   params.RegistryEndpoint,
		Username:   params.RegistryUsername,
		Password:   params.RegistryPassword,
		Namespace:  params.RegistryNamespace,
		IsReadOnly: params.RegistryIsReadOnly,
	}

	preflightpkg.InjectDefaultPreflights(kotsKinds.Preflight, kotsKinds, registrySettings)

	var preflightErr error
	defer func() {
		if preflightErr != nil {
			preflightResults := &types.PreflightResults{
				Errors: []*types.PreflightError{
					&types.PreflightError{
						Error:  preflightErr.Error(),
						IsRBAC: false,
					},
				},
			}
			if err := setPreflightResults(params.AppSlug, preflightResults); err != nil {
				logger.Error(errors.Wrap(err, "failed to set preflight results"))
				return
			}
		}
	}()

	collectors, err := registry.UpdateCollectorSpecsWithRegistryData(kotsKinds.Preflight.Spec.Collectors, registrySettings, kotsKinds.Installation, kotsKinds.License, &kotsKinds.KotsApplication)
	if err != nil {
		preflightErr = errors.Wrap(err, "failed to rewrite images in preflight")
		return preflightErr
	}
	kotsKinds.Preflight.Spec.Collectors = collectors

	go func() {
		logger.Info("preflight checks beginning",
			zap.String("appID", params.AppID),
			zap.Int64("sequence", params.NextSequence))

		setResults := func(results *types.PreflightResults) error {
			return setPreflightResults(params.AppSlug, results)
		}

		_, err := preflightpkg.Execute(kotsKinds.Preflight, false, setPreflightProgress, setResults)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to run preflight checks"))
			return
		}
	}()

	return nil
}

func setPreflightResults(appSlug string, results *types.PreflightResults) error {
	resultsBytes, err := json.Marshal(results)
	if err != nil {
		return errors.Wrap(err, "failed to marshal preflight results")
	}
	createdAt := time.Now()
	preflightData := &PreflightData{
		Result: &types.PreflightResult{
			Result:                     string(resultsBytes),
			CreatedAt:                  &createdAt,
			AppSlug:                    appSlug,
			ClusterSlug:                "this-cluster",
			Skipped:                    false,
			HasFailingStrictPreflights: hasFailingStrictPreflights(results),
		},
		Progress: "", // clear the progress once the results are set
	}
	if err := setPreflightData(preflightData); err != nil {
		return errors.Wrap(err, "failed to set preflight results")
	}
	return nil
}

func hasFailingStrictPreflights(results *types.PreflightResults) bool {
	// convert to troubleshoot type so we can use the existing function
	uploadResults := &troubleshootpreflight.UploadPreflightResults{}
	uploadResults.Results = results.Results
	for _, e := range results.Errors {
		uploadResults.Errors = append(uploadResults.Errors, &troubleshootpreflight.UploadPreflightError{
			Error: e.Error,
		})
	}
	return troubleshootpreflight.HasStrictAnalyzersFailed(uploadResults)
}

func setPreflightProgress(progress map[string]interface{}) error {
	preflightData, err := GetPreflightData()
	if err != nil {
		return errors.Wrap(err, "failed to get preflight data")
	}
	progressBytes, err := json.Marshal(progress)
	if err != nil {
		return errors.Wrap(err, "failed to marshal preflight progress")
	}
	preflightData.Progress = string(progressBytes)
	if err := setPreflightData(preflightData); err != nil {
		return errors.Wrap(err, "failed to set preflight progress")
	}
	return nil
}

func GetPreflightData() (*PreflightData, error) {
	var preflightData *PreflightData
	if _, err := os.Stat(PreflightDataFile); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "failed to stat existing preflight data")
		}
		preflightData = &PreflightData{}
	} else {
		existingBytes, err := os.ReadFile(PreflightDataFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read existing preflight data")
		}
		if err := json.Unmarshal(existingBytes, &preflightData); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal existing preflight data")
		}
	}
	return preflightData, nil
}

func setPreflightData(preflightData *PreflightData) error {
	b, err := json.Marshal(preflightData)
	if err != nil {
		return errors.Wrap(err, "failed to marshal preflight data")
	}
	if err := os.WriteFile(PreflightDataFile, b, 0644); err != nil {
		return errors.Wrap(err, "failed to write preflight data")
	}
	return nil
}

func ResetPreflightData() error {
	if err := os.RemoveAll(PreflightDataFile); err != nil {
		return errors.Wrap(err, "failed to remove preflight data")
	}
	return nil
}
