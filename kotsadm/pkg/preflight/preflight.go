package preflight

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/render"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
	"go.uber.org/zap"
)

func Run(appID string, sequence int64, archiveDir string) error {
	renderedKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return errors.Wrap(err, "failed to load rendered kots kinds")
	}

	status, err := downstream.GetDownstreamVersionStatus(appID, sequence)
	if err != nil {
		return errors.Wrapf(err, "failed to check downstream version %d status", sequence)
	}

	// preflights should not run until config is finished
	if status == "pending_config" {
		logger.Debug("not running preflights for app that is pending required configuration",
			zap.String("appID", appID),
			zap.Int64("sequence", sequence))
		return nil
	}

	if renderedKotsKinds.Preflight != nil {
		// set the status to pending_preflights
		if err := downstream.SetDownstreamVersionPendingPreflight(appID, sequence); err != nil {
			return errors.Wrapf(err, "failed to set downstream version %d pending preflight", sequence)
		}

		ignoreRBAC, err := downstream.GetIgnoreRBACErrors(appID, sequence)
		if err != nil {
			return errors.Wrap(err, "failed to get ignore rbac flag")
		}

		// render the preflight file
		// we need to convert to bytes first, so that we can reuse the renderfile function
		renderedMarshalledPreflights, err := renderedKotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Preflight")
		if err != nil {
			return errors.Wrap(err, "failed to marshal rendered preflight")
		}

		registrySettings, err := store.GetStore().GetRegistryDetailsForApp(appID)
		if err != nil {
			return errors.Wrap(err, "failed to get registry settings for app")
		}

		renderedPreflight, err := render.RenderFile(renderedKotsKinds, registrySettings, []byte(renderedMarshalledPreflights))
		if err != nil {
			return errors.Wrap(err, "failed to render preflights")
		}
		p, err := kotsutil.LoadPreflightFromContents(renderedPreflight)
		if err != nil {
			return errors.Wrap(err, "failed to load rendered preflight")
		}

		go func() {
			logger.Debug("preflight checks beginning")
			uploadPreflightResults, err := execute(appID, sequence, p, ignoreRBAC)
			if err != nil {
				err = errors.Wrap(err, "failed to run preflight checks")
				logger.Error(err)
				return
			}
			logger.Debug("preflight checks completed")

			err = maybeDeployFirstVersion(appID, sequence, uploadPreflightResults)
			if err != nil {
				err = errors.Wrap(err, "failed to deploy first version")
				logger.Error(err)
				return
			}
		}()
	} else if sequence == 0 {
		err := maybeDeployFirstVersion(appID, sequence, &troubleshootpreflight.UploadPreflightResults{})
		if err != nil {
			return errors.Wrap(err, "failed to deploy first version")
		}
	} else {
		status, err := downstream.GetDownstreamVersionStatus(appID, sequence)
		if err != nil {
			return errors.Wrap(err, "failed to get version status")
		}
		if status != "deployed" {
			if err := downstream.SetDownstreamVersionReady(appID, sequence); err != nil {
				return errors.Wrap(err, "failed to set downstream version ready")
			}
		}
	}

	return nil
}

// maybeDeployFirstVersion will deploy the first version if
// 1. preflight checks pass
// 2. we have not already deployed it
func maybeDeployFirstVersion(appID string, sequence int64, preflightResults *troubleshootpreflight.UploadPreflightResults) error {
	if sequence != 0 {
		return nil
	}

	app, err := store.GetStore().GetApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	// do not revert to first version
	if app.CurrentSequence != 0 {
		return nil
	}

	preflightState := getPreflightState(preflightResults)
	if preflightState != "pass" {
		return nil
	}

	logger.Debug("automatically deploying first app version")

	// note: this may attempt to re-deploy the first version but the operator will take care of
	// comparing the version to current

	return version.DeployVersion(appID, sequence)
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
