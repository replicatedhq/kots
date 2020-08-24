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

func GetUpdateDownloadStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

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
