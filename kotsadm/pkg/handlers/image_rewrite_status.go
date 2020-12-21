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

func (h *Handler) GetImageRewriteStatus(w http.ResponseWriter, r *http.Request) {
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
