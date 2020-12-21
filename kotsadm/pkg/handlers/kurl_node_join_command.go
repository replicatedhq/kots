package handlers

import (
	"net/http"
	"time"

	"github.com/replicatedhq/kots/kotsadm/pkg/k8s"
	"github.com/replicatedhq/kots/kotsadm/pkg/kurl"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
)

type GenerateNodeJoinCommandResponse struct {
	Command []string `json:"command"`
	Expiry  string   `json:"expiry"`
}

func (h *Handler) GenerateNodeJoinCommandWorker(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.Clientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	command, expiry, err := kurl.GenerateAddNodeCommand(client, false)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, GenerateNodeJoinCommandResponse{
		Command: command,
		Expiry:  expiry.Format(time.RFC3339),
	})
}

func (h *Handler) GenerateNodeJoinCommandMaster(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.Clientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	command, expiry, err := kurl.GenerateAddNodeCommand(client, true)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, GenerateNodeJoinCommandResponse{
		Command: command,
		Expiry:  expiry.Format(time.RFC3339),
	})
}
