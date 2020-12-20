package handlers

import (
	"net/http"

	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

type GetUpdateDownloadStatusResponse struct {
	CurrentMessage string `json:"currentMessage"`
	Status         string `json:"status"`
}

func (h *Handler) GetUpdateDownloadStatus(w http.ResponseWriter, r *http.Request) {
	status, message, err := store.GetStore().GetTaskStatus("update-download")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(err)
		return
	}

	JSON(w, http.StatusOK, GetUpdateDownloadStatusResponse{
		CurrentMessage: message,
		Status:         status,
	})
}
