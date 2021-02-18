package handlers

import (
	"net/http"
	"strings"

	snapshot "github.com/replicatedhq/kots/pkg/kotsadmsnapshot"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
)

type PingResponse struct {
	Ping                   string   `json:"ping"`
	Error                  string   `json:"error,omitempty"`
	SnapshotInProgressApps []string `json:"snapshotInProgressApps"`
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	pingResponse := PingResponse{}

	pingResponse.Ping = "pong"

	query := r.URL.Query()
	slugs := query.Get("slugs")

	if slugs != "" {
		slugsArray := strings.Split(slugs, ",")
		snapshotProgress(slugsArray, &pingResponse)
	}

	JSON(w, 200, pingResponse)
}

func snapshotProgress(slugs []string, pingResponse *PingResponse) {
	veleroStatus, err := snapshot.DetectVelero()
	if err != nil {
		logger.Error(err)
		pingResponse.Error = "failed to detect velero"
	}

	if veleroStatus == nil {
		return
	}

	for _, slug := range slugs {
		currentApp, err := store.GetStore().GetAppFromSlug(slug)
		if err != nil {
			logger.Error(err)
			pingResponse.Error = "failed to get app from app slug"
			return
		}

		backups, err := snapshot.ListBackupsForApp(currentApp.ID)
		if err != nil {
			logger.Error(err)
			pingResponse.Error = "failed to list backups"
			return
		}

		for _, backup := range backups {
			if backup.Status == "InProgress" {
				pingResponse.SnapshotInProgressApps = append(pingResponse.SnapshotInProgressApps, currentApp.Slug)
				return
			}
		}
	}
}
