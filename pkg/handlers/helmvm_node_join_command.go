package handlers

import (
	"net/http"
	"time"

	"github.com/replicatedhq/kots/pkg/helmvm"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
)

type GenerateHelmVMNodeJoinCommandResponse struct {
	Command []string `json:"command"`
	Expiry  string   `json:"expiry"`
}

func (h *Handler) GenerateHelmVMNodeJoinCommandSecondary(w http.ResponseWriter, r *http.Request) {
	client, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	command, expiry, err := helmvm.GenerateAddNodeCommand(client, false)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, GenerateHelmVMNodeJoinCommandResponse{
		Command: command,
		Expiry:  expiry.Format(time.RFC3339),
	})
}

func (h *Handler) GenerateHelmVMNodeJoinCommandPrimary(w http.ResponseWriter, r *http.Request) {
	client, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	command, expiry, err := helmvm.GenerateAddNodeCommand(client, true)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, GenerateHelmVMNodeJoinCommandResponse{
		Command: command,
		Expiry:  expiry.Format(time.RFC3339),
	})
}
