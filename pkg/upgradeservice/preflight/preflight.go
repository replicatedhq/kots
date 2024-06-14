package preflight

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	preflightpkg "github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/preflight/types"
	"github.com/replicatedhq/kots/pkg/registry"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/render"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	"github.com/replicatedhq/kots/pkg/util"
	troubleshootanalyze "github.com/replicatedhq/troubleshoot/pkg/analyze"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
	"go.uber.org/zap"
)

type PreflightData struct {
	Progress map[string]interface{}  `json:"progress"`
	Results  *types.PreflightResults `json:"results"`
}

var PreflightDataFilepath string

func init() {
	tmpDir, err := os.MkdirTemp("", "preflights")
	if err != nil {
		panic(errors.Wrap(err, "failed to create preflights data dir"))
	}
	PreflightDataFilepath = filepath.Join(tmpDir, "preflights.json")
}

func Run(app *apptypes.App, archiveDir string, sequence int64, registrySettings registrytypes.RegistrySettings, ignoreRBAC bool, reportingFn func() error) error {
	kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
	if err != nil {
		return errors.Wrap(err, "failed to load rendered kots kinds")
	}

	tsKinds, err := kotsutil.LoadTSKindsFromPath(filepath.Join(archiveDir, "rendered"))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to load troubleshoot kinds from path: %s", filepath.Join(archiveDir, "rendered")))
	}

	var preflight *troubleshootv1beta2.Preflight
	if tsKinds.PreflightsV1Beta2 != nil {
		for _, v := range tsKinds.PreflightsV1Beta2 {
			preflight = troubleshootpreflight.ConcatPreflightSpec(preflight, &v)
		}
	} else if kotsKinds.Preflight != nil {
		renderedMarshalledPreflights, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Preflight")
		if err != nil {
			return errors.Wrap(err, "failed to marshal rendered preflight")
		}
		renderedPreflight, err := render.RenderFile(rendertypes.RenderFileOptions{
			KotsKinds:        kotsKinds,
			RegistrySettings: registrySettings,
			AppSlug:          app.Slug,
			Sequence:         sequence,
			IsAirgap:         app.IsAirgap,
			Namespace:        util.PodNamespace,
			InputContent:     []byte(renderedMarshalledPreflights),
		})
		if err != nil {
			return errors.Wrap(err, "failed to render preflights")
		}
		preflight, err = kotsutil.LoadPreflightFromContents(renderedPreflight)
		if err != nil {
			return errors.Wrap(err, "failed to load rendered preflight")
		}
	}

	if preflight == nil {
		logger.Info("no preflight spec found, not running preflights")
		return nil
	}

	preflightpkg.InjectDefaultPreflights(preflight, kotsKinds, registrySettings)

	numAnalyzers := 0
	for _, analyzer := range preflight.Spec.Analyzers {
		exclude := troubleshootanalyze.GetExcludeFlag(analyzer).BoolOrDefaultFalse()
		if !exclude {
			numAnalyzers += 1
		}
	}
	if numAnalyzers == 0 {
		logger.Info("no analyzers found, not running preflights")
		return nil
	}

	var preflightErr error
	defer func() {
		if preflightErr != nil {
			if err := setPreflightResults(&types.PreflightResults{}, preflightErr); err != nil {
				logger.Error(errors.Wrap(err, "failed to set preflight results"))
				return
			}
		}
	}()

	collectors, err := registry.UpdateCollectorSpecsWithRegistryData(preflight.Spec.Collectors, registrySettings, kotsKinds.Installation, kotsKinds.License, &kotsKinds.KotsApplication)
	if err != nil {
		preflightErr = errors.Wrap(err, "failed to rewrite images in preflight")
		return preflightErr
	}
	preflight.Spec.Collectors = collectors

	go func() {
		logger.Info("preflight checks beginning",
			zap.String("appID", app.ID),
			zap.Int64("sequence", sequence))

		_, err := preflightpkg.Execute(preflight, ignoreRBAC, setPreflightProgress, setPreflightResults)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to run preflight checks"))
			return
		}

		go func() {
			if err := reportingFn(); err != nil {
				logger.Debugf("failed to report app info: %v", err)
			}
		}()
	}()

	return nil
}

func setPreflightResults(results *types.PreflightResults, runError error) error {
	preflightData, err := getPreflightData()
	if err != nil {
		return errors.Wrap(err, "failed to get preflight data")
	}
	preflightData.Results = results
	if runError != nil {
		if preflightData.Results.Errors == nil {
			preflightData.Results.Errors = []*types.PreflightError{}
		}
		preflightData.Results.Errors = append(preflightData.Results.Errors, &types.PreflightError{
			Error:  runError.Error(),
			IsRBAC: false,
		})
	}
	if err := setPreflightData(preflightData); err != nil {
		return errors.Wrap(err, "failed to set preflight results")
	}
	return nil
}

func setPreflightProgress(progress map[string]interface{}) error {
	preflightData, err := getPreflightData()
	if err != nil {
		return errors.Wrap(err, "failed to get preflight data")
	}
	preflightData.Progress = progress
	if err := setPreflightData(preflightData); err != nil {
		return errors.Wrap(err, "failed to set preflight progress")
	}
	return nil
}

func getPreflightData() (*PreflightData, error) {
	var preflightData *PreflightData
	if _, err := os.Stat(PreflightDataFilepath); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "failed to stat existing preflight data")
		}
		preflightData = &PreflightData{}
	} else {
		existingBytes, err := os.ReadFile(PreflightDataFilepath)
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
	if err := os.WriteFile(PreflightDataFilepath, b, 0644); err != nil {
		return errors.Wrap(err, "failed to write preflight data")
	}
	return nil
}
