package preflight

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/render"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	"go.uber.org/zap"
)

// Run will execute preflights
func Run(appID string, sequence int64, archiveDir string) error {
	renderedKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return errors.Wrap(err, "failed to load rendered kots kinds")
	}

	status, err := downstream.GetDownstreamVersionStatus(appID, int64(sequence))
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
		if err := downstream.SetDownstreamVersionPendingPreflight(appID, int64(sequence)); err != nil {
			return errors.Wrapf(err, "failed to set downstream version %d pending preflight", sequence)
		}

		ignoreRBAC, err := downstream.GetIgnoreRBACErrors(appID, int64(sequence))
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
			if err := execute(appID, sequence, p, ignoreRBAC); err != nil {
				logger.Error(err)
				return
			}

			logger.Debug("preflight checks completed")
		}()
	} else if sequence == 0 {
		if err := version.DeployVersion(appID, int64(sequence)); err != nil {
			return errors.Wrap(err, "failed to deploy first version")
		}
	} else {
		if err := downstream.SetDownstreamVersionReady(appID, int64(sequence)); err != nil {
			return errors.Wrap(err, "failed to set downstream version ready")
		}
	}

	return nil
}
