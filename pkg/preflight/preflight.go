package preflight

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kotsadm/pkg/logger"
)

func Run(appID string, sequence int64, archiveDir string) error {
	renderedKotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return errors.Wrap(err, "failed to load rendered kots kinds")
	}
	if renderedKotsKinds.Preflight != nil {
		// set the status to pending_preflights
		if err := app.SetDownstreamVersionPendingPreflight(appID, int64(sequence)); err != nil {
			return errors.Wrap(err, "failed to set downstream version pending preflight")
		}

		// render the preflight file
		// we need to convert to bytes first, so that we can reuse the renderfile function
		renderedMarshalledPreflights, err := renderedKotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Preflight")
		if err != nil {
			return errors.Wrap(err, "failed to marshal rendered preflight")
		}

		a, err := app.Get(appID)
		if err != nil {
			return errors.Wrap(err, "failed to get app")
		}
		renderedPreflight, err := a.RenderFile(renderedKotsKinds, []byte(renderedMarshalledPreflights))
		if err != nil {
			return errors.Wrap(err, "failed to render preflights")
		}
		p, err := kotsutil.LoadPreflightFromContents(renderedPreflight)
		if err != nil {
			return errors.Wrap(err, "failed to load rendered preflight")
		}

		go func() {
			if err := execute(appID, sequence, p); err != nil {
				logger.Error(err)
				return
			}

			logger.Debug("preflight checks completed")
		}()
	} else {
		if err := app.SetDownstreamVersionReady(appID, int64(sequence)); err != nil {
			return errors.Wrap(err, "failed to set downstream version ready")
		}
	}

	return nil
}
