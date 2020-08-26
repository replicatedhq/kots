package handlers

import (
	"net/http"

	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

type GetImageRewriteStatusResponse struct {
	Status         string `json:"status"`
	CurrentMessage string `json:"currentMessage"`
}

func GetImageRewriteStatus(w http.ResponseWriter, r *http.Request) {
	if handleOptionsRequest(w, r) {
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	status, message, err := store.GetStore().GetTaskStatus("image-rewrite")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	getImageRewriteStatusResponse := GetImageRewriteStatusResponse{
		Status:         status,
		CurrentMessage: message,
	}

	JSON(w, 200, getImageRewriteStatusResponse)
}
