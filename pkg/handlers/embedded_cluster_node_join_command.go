package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/replicatedhq/embedded-cluster/kinds/types/join"

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

	kbClient, err := h.GetKubeClient(r.Context())
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
	kbClient, err := h.GetKubeClient(r.Context())
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

	// get all the endpoints a joining node needs to ensure connectivity to
	endpoints, err := embeddedcluster.GetEndpointsToCheck(r.Context(), kbClient, roles)
	if err != nil {
		logger.Error(fmt.Errorf("failed to get the node ip addresses: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	clusterUUID, err := uuid.Parse(install.Spec.ClusterID)
	if err != nil {
		logger.Error(fmt.Errorf("failed to parse cluster id: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var currentAppVersionLabel string
	// attempt to get the current app version label from the installed app
	installedApps, err := store.GetStore().ListInstalledApps()
	if err == nil && len(installedApps) > 0 {
		// "CurrentSequence" is the latest available version of the app in a non-embedded cluster.
		// However, in an embedded cluster, the "CurrentSequence" is also the currently deployed version of the app.
		// This is because EC uses the new upgrade flow, which only creates a new app version when
		// the app version gets deployed. And because rollbacks are not supported in embedded cluster yet.
		appVersion, err := store.GetStore().GetAppVersion(installedApps[0].ID, installedApps[0].CurrentSequence)
		if err != nil {
			logger.Error(fmt.Errorf("failed to get app version: %w", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		currentAppVersionLabel = appVersion.VersionLabel
	} else {
		// if there are no installed apps, we can't get the current app version label
		logger.Info("no installed apps found")
	}

	JSON(w, http.StatusOK, join.JoinCommandResponse{
		ClusterID:              clusterUUID,
		K0sJoinCommand:         k0sJoinCommand,
		K0sToken:               k0sToken,
		EmbeddedClusterVersion: ecVersion,
		AirgapRegistryAddress:  airgapRegistryAddress,
		TCPConnectionsRequired: endpoints,
		InstallationSpec:       install.Spec,
		AppVersionLabel:        currentAppVersionLabel,
	})
}
