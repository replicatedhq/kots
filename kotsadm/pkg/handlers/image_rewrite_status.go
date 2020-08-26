package handlers

import (
	"net/http"

	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
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

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(401)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		w.WriteHeader(401)
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
