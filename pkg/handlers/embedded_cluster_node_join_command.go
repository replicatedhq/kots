package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
)

type GenerateEmbeddedClusterNodeJoinCommandResponse struct {
	Command []string `json:"command"`
}

type GetEmbeddedClusterNodeJoinCommandResponse struct {
	ClusterID      string `json:"clusterID"`
	K0sJoinCommand string `json:"k0sJoinCommand"`
	K0sToken       string `json:"k0sToken"`
}

type GenerateEmbeddedClusterNodeJoinCommandRequest struct {
	Roles []string `json:"roles"`
}

func (h *Handler) GenerateEmbeddedClusterNodeJoinCommand(w http.ResponseWriter, r *http.Request) {
	generateEmbeddedClusterNodeJoinCommandRequest := GenerateEmbeddedClusterNodeJoinCommandRequest{}
	if err := json.NewDecoder(r.Body).Decode(&generateEmbeddedClusterNodeJoinCommandRequest); err != nil {
		logger.Error(fmt.Errorf("failed to decode request body: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := store.GetStore().SetEmbeddedClusterInstallCommandRoles(generateEmbeddedClusterNodeJoinCommandRequest.Roles)
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
	nodeJoinCommand, err := embeddedcluster.GenerateAddNodeCommand(r.Context(), client, token)
	if err != nil {
		logger.Error(fmt.Errorf("failed to generate add node command: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, GenerateEmbeddedClusterNodeJoinCommandResponse{
		Command: []string{nodeJoinCommand},
	})
}

// this function relies on the token being valid for authentication
func (h *Handler) GetEmbeddedClusterNodeJoinCommand(w http.ResponseWriter, r *http.Request) {
	// read query string, ensure that the token is valid
	token := r.URL.Query().Get("token")
	roles, err := store.GetStore().GetEmbeddedClusterInstallCommandRoles(token)
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
	controllerRoleName, err := embeddedcluster.ControllerRoleName(r.Context())
	if err != nil {
		logger.Error(fmt.Errorf("failed to get controller role name: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, role := range roles {
		if role == controllerRoleName {
			k0sRole = "controller"
			break
		}
	}

	// sort roles by name, but put controller first
	roles = embeddedcluster.SortRoles(controllerRoleName, roles)

	k0sToken, err := embeddedcluster.GenerateAddNodeToken(r.Context(), client, k0sRole)
	if err != nil {
		logger.Error(fmt.Errorf("failed to generate add node token: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	k0sJoinCommand, err := embeddedcluster.GenerateK0sJoinCommand(r.Context(), client, roles)
	if err != nil {
		logger.Error(fmt.Errorf("failed to generate k0s join command: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	logger.Infof("k0s join command: %q", k0sJoinCommand)

	clusterID, err := embeddedcluster.ClusterID(client)
	if err != nil {
		logger.Error(fmt.Errorf("failed to get cluster id: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, GetEmbeddedClusterNodeJoinCommandResponse{
		ClusterID:      clusterID,
		K0sJoinCommand: k0sJoinCommand,
		K0sToken:       k0sToken,
	})
}
