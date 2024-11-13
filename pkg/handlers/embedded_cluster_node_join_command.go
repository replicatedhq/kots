package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"

	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
)

type GenerateEmbeddedClusterNodeJoinCommandResponse struct {
	Command []string `json:"command"`
}

type GetEmbeddedClusterNodeJoinCommandResponse struct {
	ClusterID              string                     `json:"clusterID"`
	K0sJoinCommand         string                     `json:"k0sJoinCommand"`
	K0sToken               string                     `json:"k0sToken"`
	EmbeddedClusterVersion string                     `json:"embeddedClusterVersion"`
	AirgapRegistryAddress  string                     `json:"airgapRegistryAddress"`
	WorkerNodeIPs          []string                   `json:"workerNodeIPs"`
	ControllerNodeIps      []string                   `json:"controllerNodeIPs"`
	InstallationSpec       ecv1beta1.InstallationSpec `json:"installationSpec,omitempty"`
}

type GenerateEmbeddedClusterNodeJoinCommandRequest struct {
	Roles []string `json:"roles"`
}

func (h *Handler) GenerateEmbeddedClusterNodeJoinCommand(w http.ResponseWriter, r *http.Request) {
	if !util.IsEmbeddedCluster() {
		logger.Errorf("not an embedded cluster")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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

	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		logger.Error(fmt.Errorf("failed to list installed apps: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(apps) == 0 {
		logger.Error(fmt.Errorf("no installed apps found"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	app := apps[0]

	kbClient, err := k8sutil.GetKubeClient(r.Context())
	if err != nil {
		logger.Error(fmt.Errorf("failed to get kubeclient: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	nodeJoinCommand, err := embeddedcluster.GenerateAddNodeCommand(r.Context(), kbClient, token, app.IsAirgap)
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
	if !util.IsEmbeddedCluster() {
		logger.Errorf("not an embedded cluster")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// read query string, ensure that the token is valid
	token := r.URL.Query().Get("token")
	roles, err := store.GetStore().GetEmbeddedClusterInstallCommandRoles(token)
	if err != nil {
		logger.Error(fmt.Errorf("failed to get k0s install command roles: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// use roles to generate join token etc
	kbClient, err := k8sutil.GetKubeClient(r.Context())
	if err != nil {
		logger.Error(fmt.Errorf("failed to get kubeclient: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	k0sRole := "worker"
	controllerRoleName, err := embeddedcluster.ControllerRoleName(r.Context(), kbClient)
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

	k0sToken, err := embeddedcluster.GenerateAddNodeToken(r.Context(), kbClient, k0sRole)
	if err != nil {
		logger.Error(fmt.Errorf("failed to generate add node token: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	k0sJoinCommand, err := embeddedcluster.GenerateK0sJoinCommand(r.Context(), kbClient, roles)
	if err != nil {
		logger.Error(fmt.Errorf("failed to generate k0s join command: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	logger.Infof("k0s join command: %q", k0sJoinCommand)

	// get the current active installation object
	install, err := embeddedcluster.GetCurrentInstallation(r.Context(), kbClient)
	if err != nil {
		logger.Error(fmt.Errorf("failed to get current install: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// extract the version from the installation object for backwards compatibility
	var ecVersion string
	if install.Spec.Config != nil {
		ecVersion = install.Spec.Config.Version
	}

	airgapRegistryAddress := ""
	if install.Spec.AirGap {
		clientset, err := k8sutil.GetClientset()
		if err != nil {
			logger.Error(fmt.Errorf("failed to get clientset: %w", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		airgapRegistryAddress, _, _ = kotsutil.GetEmbeddedRegistryCreds(clientset)
	}

	// get all the healthy node ip addresses, to be used for preflight checks in the upcoming join
	controllerNodeIPs, workerNodeIPs, err := embeddedcluster.GetAllNodeIPAddresses(r.Context(), kbClient)
	if err != nil {
		logger.Error(fmt.Errorf("failed to get the node ip addresses: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, GetEmbeddedClusterNodeJoinCommandResponse{
		ClusterID:              install.Spec.ClusterID,
		K0sJoinCommand:         k0sJoinCommand,
		K0sToken:               k0sToken,
		EmbeddedClusterVersion: ecVersion,
		AirgapRegistryAddress:  airgapRegistryAddress,
		WorkerNodeIPs:          workerNodeIPs,
		ControllerNodeIps:      controllerNodeIPs,
		InstallationSpec:       install.Spec,
	})
}
