package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/tasks"
)

type GetUpdateDownloadStatusResponse struct {
	CurrentMessage string `json:"currentMessage"`
	Status         string `json:"status"`
}

func (h *Handler) GetUpdateDownloadStatus(w http.ResponseWriter, r *http.Request) {
	status, message, err := tasks.GetTaskStatus("update-download")
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

type GetAppVersionDownloadStatusResponse struct {
	CurrentMessage string `json:"currentMessage"`
	Status         string `json:"status"`
	Error          string `json:"error"`
}

func (h *Handler) GetAppVersionDownloadStatus(w http.ResponseWriter, r *http.Request) {
	getAppVersionDownloadStatusResponse := GetAppVersionDownloadStatusResponse{}

	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		errMsg := "failed to parse sequence number"
		logger.Error(errors.Wrap(err, errMsg))
		getAppVersionDownloadStatusResponse.Error = errMsg
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	taskID := fmt.Sprintf("update-download.%d", sequence)
	status, message, err := tasks.GetTaskStatus(taskID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get %s task status", taskID)
		logger.Error(errors.Wrap(err, errMsg))
		getAppVersionDownloadStatusResponse.Error = errMsg
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	getAppVersionDownloadStatusResponse.CurrentMessage = message
	getAppVersionDownloadStatusResponse.Status = status

	JSON(w, http.StatusOK, getAppVersionDownloadStatusResponse)
}
