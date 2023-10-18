package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/replicatedhq/kots/pkg/helmvm"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store/kotsstore"
)

type GenerateK0sNodeJoinCommandResponse struct {
	Command []string `json:"command"`
}

type GetK0sNodeJoinCommandResponse struct {
	ClusterID      string `json:"clusterID"`
	K0sJoinCommand string `json:"k0sJoinCommand"`
	K0sToken       string `json:"k0sToken"`
}

type GenerateHelmVMNodeJoinCommandRequest struct {
	Roles []string `json:"roles"`
}

func (h *Handler) GenerateK0sNodeJoinCommand(w http.ResponseWriter, r *http.Request) {
	generateHelmVMNodeJoinCommandRequest := GenerateHelmVMNodeJoinCommandRequest{}
	if err := json.NewDecoder(r.Body).Decode(&generateHelmVMNodeJoinCommandRequest); err != nil {
		logger.Error(fmt.Errorf("failed to decode request body: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	store := kotsstore.StoreFromEnv()
	token, err := store.SetK0sInstallCommandRoles(generateHelmVMNodeJoinCommandRequest.Roles)
	if err != nil {
		logger.Error(fmt.Errorf("failed to set k0s install command roles: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	client, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(fmt.Errorf("failed to get clientset: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	nodeJoinCommand, err := helmvm.GenerateAddNodeCommand(r.Context(), client, token)
	if err != nil {
		logger.Error(fmt.Errorf("failed to generate add node command: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, GenerateK0sNodeJoinCommandResponse{
		Command: []string{nodeJoinCommand},
	})
}

// this function relies on the token being valid for authentication
func (h *Handler) GetK0sNodeJoinCommand(w http.ResponseWriter, r *http.Request) {
	// read query string, ensure that the token is valid
	token := r.URL.Query().Get("token")
	store := kotsstore.StoreFromEnv()
	roles, err := store.GetK0sInstallCommandRoles(token)
	if err != nil {
		logger.Error(fmt.Errorf("failed to get k0s install command roles: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// use roles to generate join token etc
	client, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(fmt.Errorf("failed to get clientset: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	k0sRole := "worker"
	for _, role := range roles {
		if role == "controller" {
			k0sRole = "controller"
			break
		}
	}

	k0sToken, err := helmvm.GenerateAddNodeToken(r.Context(), client, k0sRole)
	if err != nil {
		logger.Error(fmt.Errorf("failed to generate add node token: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	k0sJoinCommand, err := helmvm.GenerateK0sJoinCommand(r.Context(), client, roles)
	if err != nil {
		logger.Error(fmt.Errorf("failed to generate k0s join command: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	clusterID, err := helmvm.ClusterID(client)
	if err != nil {
		logger.Error(fmt.Errorf("failed to get cluster id: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, GetK0sNodeJoinCommandResponse{
		ClusterID:      clusterID,
		K0sJoinCommand: k0sJoinCommand,
		K0sToken:       k0sToken,
	})
}
