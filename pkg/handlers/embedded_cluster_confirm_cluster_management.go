package handlers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
)

type ConfirmEmbeddedClusterManagementResponse struct {
	VersionStatus string `json:"versionStatus"`
}

func (h *Handler) ConfirmEmbeddedClusterManagement(w http.ResponseWriter, r *http.Request) {
	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		logger.Error(fmt.Errorf("failed to list installed apps: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(apps) == 0 {
		logger.Error(fmt.Errorf("no installed apps found"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	app := apps[0]

	downstreamVersions, err := store.GetStore().FindDownstreamVersions(app.ID, true)
	if err != nil {
		logger.Error(fmt.Errorf("failed to find downstream versions: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(downstreamVersions.PendingVersions) == 0 {
		logger.Error(fmt.Errorf("no pending versions found"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	pendingVersion := downstreamVersions.PendingVersions[0]

	if pendingVersion.Status == storetypes.VersionPendingClusterManagement {
		archiveDir, err := os.MkdirTemp("", "kotsadm")
		if err != nil {
			logger.Error(fmt.Errorf("failed to create temp dir: %w", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(archiveDir)

		err = store.GetStore().GetAppVersionArchive(app.ID, pendingVersion.Sequence, archiveDir)
		if err != nil {
			logger.Error(fmt.Errorf("failed to get app version archive: %w", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
		if err != nil {
			logger.Error(fmt.Errorf("failed to load kots kinds: %w", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		downstreamVersionStatus := storetypes.VersionPending
		if kotsKinds.IsConfigurable() {
			downstreamVersionStatus = storetypes.VersionPendingConfig
		} else if kotsKinds.HasPreflights() {
			downstreamVersionStatus = storetypes.VersionPendingPreflight
			if err := preflight.Run(app.ID, app.Slug, pendingVersion.Sequence, false, archiveDir); err != nil {
				logger.Error(errors.Wrap(err, "failed to start preflights"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		pendingVersion.Status = downstreamVersionStatus

		if err := store.GetStore().SetDownstreamVersionStatus(app.ID, pendingVersion.Sequence, pendingVersion.Status, ""); err != nil {
			logger.Error(fmt.Errorf("failed to set downstream version status: %w", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	JSON(w, http.StatusOK, ConfirmEmbeddedClusterManagementResponse{
		VersionStatus: string(pendingVersion.Status),
	})
}
