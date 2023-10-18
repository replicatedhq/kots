package handlers

import (
	"encoding/json"
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

type GenerateHelmVMNodeJoinCommandRequest struct {
	Roles []string `json:"roles"`
}

func (h *Handler) GenerateHelmVMNodeJoinCommand(w http.ResponseWriter, r *http.Request) {
	generateHelmVMNodeJoinCommandRequest := GenerateHelmVMNodeJoinCommandRequest{}
	if err := json.NewDecoder(r.Body).Decode(&generateHelmVMNodeJoinCommandRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	client, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	k0sRole := "worker"
	for _, role := range generateHelmVMNodeJoinCommandRequest.Roles {
		if role == "controller" {
			k0sRole = "controller"
			break
		}
	}

	command, expiry, err := helmvm.GenerateAddNodeCommand(r.Context(), client, k0sRole)
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
