package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/handlers/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	snapshot "github.com/replicatedhq/kots/pkg/kotsadmsnapshot"
	snapshottypes "github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/print"
	kotssnapshot "github.com/replicatedhq/kots/pkg/snapshot"
	kotssnapshottypes "github.com/replicatedhq/kots/pkg/snapshot/types"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/robfig/cron"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
)

const (
	urlPattern = `\b(https?):\/\/[\-A-Za-z0-9+&@#\/%?=~_|!:,.;]*[\-A-Za-z0-9+&@#\/%=~_|]`
)

type GlobalSnapshotSettingsResponse struct {
	VeleroVersion      string   `json:"veleroVersion"`
	VeleroPlugins      []string `json:"veleroPlugins"`
	VeleroNamespace    string   `json:"veleroNamespace"`
	IsVeleroRunning    bool     `json:"isVeleroRunning"`
	IsMinioDisabled    bool     `json:"isMinioDisabled"`
	VeleroPod          string   `json:"veleroPod"`
	NodeAgentVersion   string   `json:"nodeAgentVersion"`
	IsNodeAgentRunning bool     `json:"isNodeAgentRunning"`
	NodeAgentPods      []string `json:"nodeAgentPods"`

	KotsadmNamespace     string `json:"kotsadmNamespace"`
	IsKurl               bool   `json:"isKurl"`
	IsMinimalRBACEnabled bool   `json:"isMinimalRBACEnabled"`

	Store            *kotssnapshottypes.Store            `json:"store,omitempty"`
	FileSystemConfig *kotssnapshottypes.FileSystemConfig `json:"fileSystemConfig,omitempty"`

	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type UpdateGlobalSnapshotSettingsRequest struct {
	Provider string `json:"provider"`
	Bucket   string `json:"bucket"`
	Path     string `json:"path"`

	AWS        *kotssnapshottypes.StoreAWS    `json:"aws"`
	Google     *kotssnapshottypes.StoreGoogle `json:"gcp"`
	Azure      *kotssnapshottypes.StoreAzure  `json:"azure"`
	Other      *kotssnapshottypes.StoreOther  `json:"other"`
	Internal   bool                           `json:"internal"`
	FileSystem *FileSystemOptions             `json:"fileSystem"`

	CACertData []byte `json:"caCertData"`
}

type GetFileSystemSnapshotProviderInstructionsResponse struct {
	Success      bool                                  `json:"success"`
	Error        string                                `json:"error,omitempty"`
	Instructions []print.VeleroInstallationInstruction `json:"instructions,omitempty"`
}

type GetFileSystemSnapshotProviderInstructionsRequest struct {
	FileSystemOptions FileSystemOptions `json:"fileSystemOptions"`
}

type FileSystemOptions struct {
	kotssnapshottypes.FileSystemConfig `json:",inline"`
	ForceReset                         bool `json:"forceReset,omitempty"`
}

type SnapshotConfig struct {
	AutoEnabled  bool                            `json:"autoEnabled"`
	AutoSchedule *snapshottypes.SnapshotSchedule `json:"autoSchedule"`
	TTl          *snapshottypes.SnapshotTTL      `json:"ttl"`
}

type VeleroStatus struct {
	IsVeleroInstalled bool `json:"isVeleroInstalled"`
}

func (h *Handler) UpdateGlobalSnapshotSettings(w http.ResponseWriter, r *http.Request) {
	globalSnapshotSettingsResponse := GlobalSnapshotSettingsResponse{
		Success: false,
	}

	// check minimal rbac
	if err := requiresKotsadmVeleroAccess(w, r); err != nil {
		return
	}

	updateGlobalSnapshotSettingsRequest := UpdateGlobalSnapshotSettingsRequest{}
	if err := json.NewDecoder(r.Body).Decode(&updateGlobalSnapshotSettingsRequest); err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to decode request body"
		JSON(w, http.StatusBadRequest, globalSnapshotSettingsResponse)
		return
	}

	// Validate Endpoint - velero is picky
	if updateGlobalSnapshotSettingsRequest.Other != nil {

		urlRe := regexp.MustCompile(urlPattern)

		if !urlRe.Match([]byte(updateGlobalSnapshotSettingsRequest.Other.Endpoint)) {
			globalSnapshotSettingsResponse.Error = "invalid endpoint for S3 compatible storage"
			JSON(w, http.StatusUnprocessableEntity, globalSnapshotSettingsResponse)
			return
		}
	}

	kotsadmNamespace := util.PodNamespace

	isMinioDisabled, err := kotssnapshot.IsFileSystemMinioDisabled(kotsadmNamespace)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to check if file system minio is disabled"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}

	veleroStatus, err := kotssnapshot.DetectVelero(r.Context(), kotsadmNamespace)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to detect velero"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}
	if veleroStatus == nil {
		JSON(w, http.StatusOK, globalSnapshotSettingsResponse)
		return
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to create k8s clientset"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}

	isKurl, err := kurl.IsKurl(clientset)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to check if cluster is kurl"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}

	globalSnapshotSettingsResponse.VeleroVersion = veleroStatus.Version
	globalSnapshotSettingsResponse.VeleroPlugins = veleroStatus.Plugins
	globalSnapshotSettingsResponse.VeleroNamespace = veleroStatus.Namespace
	globalSnapshotSettingsResponse.VeleroPod = veleroStatus.VeleroPod
	globalSnapshotSettingsResponse.IsVeleroRunning = veleroStatus.Status == "Ready"
	globalSnapshotSettingsResponse.NodeAgentVersion = veleroStatus.NodeAgentVersion
	globalSnapshotSettingsResponse.IsNodeAgentRunning = veleroStatus.NodeAgentStatus == "Ready"
	globalSnapshotSettingsResponse.NodeAgentPods = veleroStatus.NodeAgentPods
	globalSnapshotSettingsResponse.KotsadmNamespace = kotsadmNamespace
	globalSnapshotSettingsResponse.IsKurl = isKurl
	globalSnapshotSettingsResponse.IsMinimalRBACEnabled = !k8sutil.IsKotsadmClusterScoped(r.Context(), clientset, kotsadmNamespace)
	globalSnapshotSettingsResponse.IsMinioDisabled = isMinioDisabled

	registryConfig, err := kotsadm.GetRegistryConfigFromCluster(kotsadmNamespace, clientset)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to get kotsadm options from cluster"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}

	if updateGlobalSnapshotSettingsRequest.FileSystem != nil {
		if !globalSnapshotSettingsResponse.IsMinioDisabled {
			// make sure the file system provider is configured and deployed first
			if err := configureMinioFileSystemProvider(r.Context(), clientset, kotsadmNamespace, registryConfig, *updateGlobalSnapshotSettingsRequest.FileSystem); err != nil {
				if _, ok := errors.Cause(err).(*kotssnapshot.ResetFileSystemError); ok {
					globalSnapshotSettingsResponse.Error = err.Error()
					JSON(w, http.StatusConflict, globalSnapshotSettingsResponse)
					return
				}
				if _, ok := errors.Cause(err).(*kotssnapshot.HostPathNotFoundError); ok {
					globalSnapshotSettingsResponse.Error = err.Error()
					JSON(w, http.StatusBadRequest, globalSnapshotSettingsResponse)
					return
				}
				if err, ok := errors.Cause(err).(util.ActionableError); ok {
					globalSnapshotSettingsResponse.Error = err.Error()
					JSON(w, http.StatusBadRequest, globalSnapshotSettingsResponse)
					return
				}

				logger.Error(err)
				globalSnapshotSettingsResponse.Error = "failed to configure file system provider"
				JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
				return
			}
		} else {
			if err := configureLvpFileSystemProvider(r.Context(), clientset, kotsadmNamespace, registryConfig, *updateGlobalSnapshotSettingsRequest.FileSystem); err != nil {
				globalSnapshotSettingsResponse.Error = err.Error()
				JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
				return
			}
		}
	}

	var filesystem *kotssnapshottypes.FileSystemConfig
	if updateGlobalSnapshotSettingsRequest.FileSystem != nil {
		filesystem = &updateGlobalSnapshotSettingsRequest.FileSystem.FileSystemConfig
	}

	// update/configure store
	options := kotssnapshot.ConfigureStoreOptions{
		Provider: updateGlobalSnapshotSettingsRequest.Provider,
		Bucket:   updateGlobalSnapshotSettingsRequest.Bucket,
		Path:     updateGlobalSnapshotSettingsRequest.Path,

		AWS:        updateGlobalSnapshotSettingsRequest.AWS,
		Google:     updateGlobalSnapshotSettingsRequest.Google,
		Azure:      updateGlobalSnapshotSettingsRequest.Azure,
		Other:      updateGlobalSnapshotSettingsRequest.Other,
		Internal:   updateGlobalSnapshotSettingsRequest.Internal,
		FileSystem: filesystem,

		KotsadmNamespace: kotsadmNamespace,
		RegistryConfig:   &registryConfig,
		IsMinioDisabled:  globalSnapshotSettingsResponse.IsMinioDisabled,

		CACertData: updateGlobalSnapshotSettingsRequest.CACertData,
	}
	updatedStore, err := kotssnapshot.ConfigureStore(r.Context(), options)
	if err != nil {
		if _, ok := errors.Cause(err).(*kotssnapshot.InvalidStoreDataError); ok {
			logger.Error(err)
			globalSnapshotSettingsResponse.Error = fmt.Sprintf("invalid store data: %s", err)
			JSON(w, http.StatusBadRequest, globalSnapshotSettingsResponse)
			return
		}
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to configure store"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}

	if updatedStore.FileSystem != nil {
		fileSystemConfig, err := kotssnapshot.GetCurrentFileSystemConfig(r.Context(), kotsadmNamespace, globalSnapshotSettingsResponse.IsMinioDisabled)
		if err != nil {
			logger.Error(err)
			globalSnapshotSettingsResponse.Error = "failed to get file system config"
			JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
			return
		}
		globalSnapshotSettingsResponse.FileSystemConfig = fileSystemConfig
	}

	globalSnapshotSettingsResponse.Store = updatedStore
	globalSnapshotSettingsResponse.Success = true

	JSON(w, http.StatusOK, globalSnapshotSettingsResponse)
}

func (h *Handler) GetGlobalSnapshotSettings(w http.ResponseWriter, r *http.Request) {
	globalSnapshotSettingsResponse := GlobalSnapshotSettingsResponse{
		Success: false,
	}

	// check minimal rbac
	if err := requiresKotsadmVeleroAccess(w, r); err != nil {
		return
	}

	kotsadmNamespace := util.PodNamespace

	isMinioDisabled, err := kotssnapshot.IsFileSystemMinioDisabled(kotsadmNamespace)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to check if file system minio is disabled"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}

	veleroStatus, err := kotssnapshot.DetectVelero(r.Context(), kotsadmNamespace)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to detect velero"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}
	if veleroStatus == nil {
		JSON(w, http.StatusOK, globalSnapshotSettingsResponse)
		return
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to create k8s clientset"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}

	isKurl, err := kurl.IsKurl(clientset)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to check if cluster is kurl"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}

	globalSnapshotSettingsResponse.VeleroVersion = veleroStatus.Version
	globalSnapshotSettingsResponse.VeleroPlugins = veleroStatus.Plugins
	globalSnapshotSettingsResponse.VeleroNamespace = veleroStatus.Namespace
	globalSnapshotSettingsResponse.VeleroPod = veleroStatus.VeleroPod
	globalSnapshotSettingsResponse.IsVeleroRunning = veleroStatus.Status == "Ready"
	globalSnapshotSettingsResponse.NodeAgentVersion = veleroStatus.NodeAgentVersion
	globalSnapshotSettingsResponse.IsNodeAgentRunning = veleroStatus.NodeAgentStatus == "Ready"
	globalSnapshotSettingsResponse.NodeAgentPods = veleroStatus.NodeAgentPods
	globalSnapshotSettingsResponse.KotsadmNamespace = kotsadmNamespace
	globalSnapshotSettingsResponse.IsKurl = isKurl
	globalSnapshotSettingsResponse.IsMinimalRBACEnabled = !k8sutil.IsKotsadmClusterScoped(r.Context(), clientset, kotsadmNamespace)
	globalSnapshotSettingsResponse.IsMinioDisabled = isMinioDisabled

	store, err := kotssnapshot.GetGlobalStore(r.Context(), kotsadmNamespace, nil)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to get store"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}

	if store != nil {
		if err := kotssnapshot.Redact(store); err != nil {
			logger.Error(err)
			globalSnapshotSettingsResponse.Error = "failed to redact"
			JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
			return
		}

		if store.FileSystem != nil {
			fileSystemConfig, err := kotssnapshot.GetCurrentFileSystemConfig(r.Context(), kotsadmNamespace, globalSnapshotSettingsResponse.IsMinioDisabled)
			if err != nil {
				logger.Error(err)
				globalSnapshotSettingsResponse.Error = "failed to get file system config"
				JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
				return
			}
			globalSnapshotSettingsResponse.FileSystemConfig = fileSystemConfig
		}
	}

	globalSnapshotSettingsResponse.Store = store
	globalSnapshotSettingsResponse.Success = true

	JSON(w, http.StatusOK, globalSnapshotSettingsResponse)
}

func (h *Handler) GetFileSystemSnapshotProviderInstructions(w http.ResponseWriter, r *http.Request) {
	response := GetFileSystemSnapshotProviderInstructionsResponse{
		Success: false,
	}

	request := GetFileSystemSnapshotProviderInstructionsRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		errMsg := "failed to decode request body"
		logger.Error(errors.Wrap(err, errMsg))
		response.Error = errMsg
		JSON(w, http.StatusBadRequest, response)
		return
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		errMsg := "failed to get k8s client set"
		response.Error = errMsg
		logger.Error(errors.Wrap(err, errMsg))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	kotsadmNamespace := util.PodNamespace

	registryConfig, err := kotsadm.GetRegistryConfigFromCluster(kotsadmNamespace, clientset)
	if err != nil {
		errMsg := "failed to get kotsadm options from cluster"
		response.Error = errMsg
		logger.Error(errors.Wrap(err, errMsg))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	configureCommand := ""
	if request.FileSystemOptions.HostPath != nil {
		configureCommand = fmt.Sprintf(`kubectl kots velero configure-hostpath --hostpath %s`, *request.FileSystemOptions.HostPath)
	} else if request.FileSystemOptions.NFS != nil {
		configureCommand = fmt.Sprintf(`kubectl kots velero configure-nfs --nfs-server %s --nfs-path %s`, request.FileSystemOptions.NFS.Server, request.FileSystemOptions.NFS.Path)
	}

	configureCommand += fmt.Sprintf(` --namespace %s`, kotsadmNamespace)

	if request.FileSystemOptions.ForceReset {
		configureCommand += " --force-reset"
	}

	if registryConfig.OverrideRegistry != "" {
		configureCommand += fmt.Sprintf(` --kotsadm-registry %s`, registryConfig.OverrideRegistry)

		if registryConfig.OverrideNamespace != "" {
			configureCommand += fmt.Sprintf(` --kotsadm-namespace %s`, registryConfig.OverrideNamespace)
		}
		if registryConfig.Username != "" {
			configureCommand += fmt.Sprintf(` --registry-username %s`, registryConfig.Username)
		}
		if registryConfig.Password != "" {
			configureCommand += fmt.Sprintf(` --registry-password %s`, registryConfig.Password)
		}
	}

	isMinioDisabled, err := kotssnapshot.IsFileSystemMinioDisabled(kotsadmNamespace)
	if err != nil {
		logger.Error(err)
		response.Error = "failed to check if file system minio is disabled"
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	var plugin kotssnapshottypes.VeleroPlugin
	if isMinioDisabled {
		plugin = kotssnapshottypes.VeleroLVPPlugin
	} else {
		plugin = kotssnapshottypes.VeleroAWSPlugin
	}

	response.Success = true
	response.Instructions = print.VeleroInstallationInstructionsForUI(plugin, &registryConfig, configureCommand)

	JSON(w, http.StatusOK, response)
}

func configureLvpFileSystemProvider(ctx context.Context, clientset kubernetes.Interface, namespace string, registryConfig kotsadmtypes.RegistryConfig, fileSystemOptions FileSystemOptions) error {
	deployOptions := kotssnapshot.FileSystemDeployOptions{
		Namespace:        namespace,
		IsOpenShift:      k8sutil.IsOpenShift(clientset),
		ForceReset:       fileSystemOptions.ForceReset,
		FileSystemConfig: fileSystemOptions.FileSystemConfig,
	}
	if err := kotssnapshot.DeployFileSystemLvp(ctx, clientset, deployOptions, registryConfig); err != nil {
		return err
	}
	return nil
}

func configureMinioFileSystemProvider(ctx context.Context, clientset kubernetes.Interface, namespace string, registryConfig kotsadmtypes.RegistryConfig, fileSystemOptions FileSystemOptions) error {
	deployOptions := kotssnapshot.FileSystemDeployOptions{
		Namespace:        namespace,
		IsOpenShift:      k8sutil.IsOpenShift(clientset),
		ForceReset:       fileSystemOptions.ForceReset,
		FileSystemConfig: fileSystemOptions.FileSystemConfig,
	}

	if err := kotssnapshot.DeployFileSystemMinio(ctx, clientset, deployOptions, registryConfig); err != nil {
		return errors.Wrap(err, "failed to deploy file system minio")
	}

	err := k8sutil.WaitForDeploymentReady(ctx, clientset, namespace, kotssnapshot.FileSystemMinioDeploymentName, time.Minute*3)
	if err != nil {
		return errors.Wrap(err, "failed to wait for file system minio")
	}

	err = kotssnapshot.CreateFileSystemMinioBucket(ctx, clientset, namespace, registryConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create default bucket")
	}

	return nil
}

func (h *Handler) GetSnapshotConfig(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ttl := &snapshottypes.SnapshotTTL{}
	if foundApp.SnapshotTTL != "" {
		parsedTTL, err := snapshot.ParseTTL(foundApp.SnapshotTTL)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ttl.InputValue = strconv.FormatInt(parsedTTL.Quantity, 10)
		ttl.InputTimeUnit = parsedTTL.Unit
		ttl.Converted = foundApp.SnapshotTTL
	} else {
		ttl.InputValue = "1"
		ttl.InputTimeUnit = "month"
		ttl.Converted = "720h"
	}

	snapshotSchedule := &snapshottypes.SnapshotSchedule{}
	if foundApp.SnapshotSchedule != "" {
		snapshotSchedule.Schedule = foundApp.SnapshotSchedule
	} else {
		snapshotSchedule.Schedule = "0 0 * * MON"
	}

	getSnapshotConfigResponse := SnapshotConfig{}
	getSnapshotConfigResponse.AutoEnabled = foundApp.SnapshotSchedule != ""
	getSnapshotConfigResponse.AutoSchedule = snapshotSchedule
	getSnapshotConfigResponse.TTl = ttl

	JSON(w, http.StatusOK, getSnapshotConfigResponse)
}

func (h *Handler) GetVeleroStatus(w http.ResponseWriter, r *http.Request) {
	getVeleroStatusResponse := VeleroStatus{}

	detectVelero, err := kotssnapshot.DetectVelero(r.Context(), util.PodNamespace)
	if err != nil {
		logger.Error(err)
		getVeleroStatusResponse.IsVeleroInstalled = false
		JSON(w, http.StatusInternalServerError, getVeleroStatusResponse)
		return
	}

	if detectVelero == nil {
		getVeleroStatusResponse.IsVeleroInstalled = false
		JSON(w, http.StatusOK, getVeleroStatusResponse)
		return
	}

	getVeleroStatusResponse.IsVeleroInstalled = true
	JSON(w, http.StatusOK, getVeleroStatusResponse)
}

type SaveSnapshotScheduleRequest struct {
	AppID       string `json:"appId"`
	Schedule    string `json:"schedule"`
	AutoEnabled bool   `json:"autoEnabled"`
}

type SaveSnapshotConfigResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) SaveSnapshotSchedule(w http.ResponseWriter, r *http.Request) {
	responseBody := SaveSnapshotConfigResponse{}

	// check minimal rbac
	if err := requiresKotsadmVeleroAccess(w, r); err != nil {
		return
	}

	requestBody := SaveSnapshotScheduleRequest{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		logger.Error(err)
		responseBody.Error = "failed to decode request body"
		JSON(w, http.StatusBadRequest, responseBody)
		return
	}

	app, err := store.GetStore().GetApp(requestBody.AppID)
	if err != nil {
		logger.Error(err)
		responseBody.Error = "Failed to get app"
		JSON(w, http.StatusInternalServerError, responseBody)
		return
	}

	if !requestBody.AutoEnabled {
		if err := store.GetStore().SetSnapshotSchedule(app.ID, ""); err != nil {
			logger.Error(err)
			responseBody.Error = "Failed to clear snapshot schedule"
			JSON(w, http.StatusInternalServerError, responseBody)
			return
		}
		if err := store.GetStore().DeletePendingScheduledSnapshots(app.ID); err != nil {
			logger.Error(err)
			responseBody.Error = "Failed to delete scheduled snapshots"
			JSON(w, http.StatusInternalServerError, responseBody)
			return
		}
		responseBody.Success = true
		JSON(w, http.StatusOK, responseBody)
		return
	}

	cronSchedule, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(requestBody.Schedule)
	if err != nil {
		logger.Error(err)
		responseBody.Error = fmt.Sprintf("Invalid cron schedule expression: %s", requestBody.Schedule)
		JSON(w, http.StatusBadRequest, responseBody)
		return
	}

	if requestBody.Schedule != app.SnapshotSchedule {
		if err := store.GetStore().DeletePendingScheduledSnapshots(app.ID); err != nil {
			logger.Error(err)
			responseBody.Error = "Failed to delete scheduled snapshots"
			JSON(w, http.StatusInternalServerError, responseBody)
			return
		}
		if err := store.GetStore().SetSnapshotSchedule(app.ID, requestBody.Schedule); err != nil {
			logger.Error(err)
			responseBody.Error = "Failed to save snapshot schedule"
			JSON(w, http.StatusInternalServerError, responseBody)
			return
		}
		queued := cronSchedule.Next(time.Now())
		id := strings.ToLower(rand.String(32))
		if err := store.GetStore().CreateScheduledSnapshot(id, app.ID, queued); err != nil {
			logger.Error(err)
			responseBody.Error = "Failed to create first scheduled snapshot"
			JSON(w, http.StatusInternalServerError, responseBody)
			return
		}
	}

	responseBody.Success = true
	JSON(w, http.StatusOK, responseBody)
}

type SaveSnapshotRetentionRequest struct {
	AppID         string `json:"appId"`
	InputValue    string `json:"inputValue"`
	InputTimeUnit string `json:"inputTimeUnit"`
}

func (h *Handler) SaveSnapshotRetention(w http.ResponseWriter, r *http.Request) {
	responseBody := SaveSnapshotConfigResponse{}

	// check minimal rbac
	if err := requiresKotsadmVeleroAccess(w, r); err != nil {
		return
	}

	requestBody := SaveSnapshotRetentionRequest{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		logger.Error(err)
		responseBody.Error = "failed to decode request body"
		JSON(w, http.StatusBadRequest, responseBody)
		return
	}

	app, err := store.GetStore().GetApp(requestBody.AppID)
	if err != nil {
		logger.Error(err)
		responseBody.Error = "Failed to get app"
		JSON(w, http.StatusInternalServerError, responseBody)
		return
	}

	retention, err := snapshot.FormatTTL(requestBody.InputValue, requestBody.InputTimeUnit)
	if err != nil {
		logger.Error(err)
		responseBody.Error = fmt.Sprintf("Invalid snapshot retention: %s %s", requestBody.InputValue, requestBody.InputTimeUnit)
		JSON(w, http.StatusBadRequest, responseBody)
		return
	}

	if app.SnapshotTTL != retention {
		app.SnapshotTTL = retention
		if err := store.GetStore().SetSnapshotTTL(app.ID, retention); err != nil {
			logger.Error(err)
			responseBody.Error = "Failed to set snapshot retention"
			JSON(w, http.StatusInternalServerError, responseBody)
			return
		}
	}

	responseBody.Success = true
	JSON(w, http.StatusOK, responseBody)
}

type InstanceSnapshotConfig struct {
	AutoEnabled  bool                            `json:"autoEnabled"`
	AutoSchedule *snapshottypes.SnapshotSchedule `json:"autoSchedule"`
	TTl          *snapshottypes.SnapshotTTL      `json:"ttl"`
}

func (h *Handler) GetInstanceSnapshotConfig(w http.ResponseWriter, r *http.Request) {
	clusters, err := store.GetStore().ListClusters()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(clusters) == 0 {
		logger.Error(errors.New("No clusters found"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	c := clusters[0]

	ttl := &snapshottypes.SnapshotTTL{}
	if c.SnapshotTTL != "" {
		parsedTTL, err := snapshot.ParseTTL(c.SnapshotTTL)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ttl.InputValue = strconv.FormatInt(parsedTTL.Quantity, 10)
		ttl.InputTimeUnit = parsedTTL.Unit
		ttl.Converted = c.SnapshotTTL
	} else {
		ttl.InputValue = "1"
		ttl.InputTimeUnit = "month"
		ttl.Converted = "720h"
	}

	snapshotSchedule := &snapshottypes.SnapshotSchedule{}
	if c.SnapshotSchedule != "" {
		snapshotSchedule.Schedule = c.SnapshotSchedule
	} else {
		snapshotSchedule.Schedule = "0 0 * * MON"
	}

	getInstanceSnapshotConfigResponse := InstanceSnapshotConfig{}
	getInstanceSnapshotConfigResponse.AutoEnabled = c.SnapshotSchedule != ""
	getInstanceSnapshotConfigResponse.AutoSchedule = snapshotSchedule
	getInstanceSnapshotConfigResponse.TTl = ttl

	JSON(w, http.StatusOK, getInstanceSnapshotConfigResponse)
}

type SaveInstanceSnapshotScheduleRequest struct {
	Schedule    string `json:"schedule"`
	AutoEnabled bool   `json:"autoEnabled"`
}

type SaveInstanceSnapshotConfigResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) SaveInstanceSnapshotSchedule(w http.ResponseWriter, r *http.Request) {
	responseBody := SaveInstanceSnapshotConfigResponse{}

	// check minimal rbac
	if err := requiresKotsadmVeleroAccess(w, r); err != nil {
		return
	}

	requestBody := SaveInstanceSnapshotScheduleRequest{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		logger.Error(err)
		responseBody.Error = "failed to decode request body"
		JSON(w, http.StatusBadRequest, responseBody)
		return
	}

	clusters, err := store.GetStore().ListClusters()
	if err != nil {
		logger.Error(err)
		responseBody.Error = "Failed to list clusters"
		JSON(w, http.StatusInternalServerError, responseBody)
		return
	}
	if len(clusters) == 0 {
		err := errors.New("No clusters found")
		logger.Error(err)
		responseBody.Error = err.Error()
		JSON(w, http.StatusInternalServerError, responseBody)
		return
	}
	c := clusters[0]

	if !requestBody.AutoEnabled {
		if err := store.GetStore().SetInstanceSnapshotSchedule(c.ClusterID, ""); err != nil {
			logger.Error(err)
			responseBody.Error = "Failed to clear instance snapshot schedule"
			JSON(w, http.StatusInternalServerError, responseBody)
			return
		}
		if err := store.GetStore().DeletePendingScheduledInstanceSnapshots(c.ClusterID); err != nil {
			logger.Error(err)
			responseBody.Error = "Failed to delete pending scheduled instance snapshots"
			JSON(w, http.StatusInternalServerError, responseBody)
			return
		}
		responseBody.Success = true
		JSON(w, http.StatusOK, responseBody)
		return
	}

	cronSchedule, err := cron.ParseStandard(requestBody.Schedule)
	if err != nil {
		logger.Error(err)
		responseBody.Error = fmt.Sprintf("Invalid cron schedule expression: %s", requestBody.Schedule)
		JSON(w, http.StatusBadRequest, responseBody)
		return
	}

	if requestBody.Schedule != c.SnapshotSchedule {
		if err := store.GetStore().DeletePendingScheduledInstanceSnapshots(c.ClusterID); err != nil {
			logger.Error(err)
			responseBody.Error = "Failed to delete scheduled snapshots"
			JSON(w, http.StatusInternalServerError, responseBody)
			return
		}
		if err := store.GetStore().SetInstanceSnapshotSchedule(c.ClusterID, requestBody.Schedule); err != nil {
			logger.Error(err)
			responseBody.Error = "Failed to save instance snapshot schedule"
			JSON(w, http.StatusInternalServerError, responseBody)
			return
		}
		queued := cronSchedule.Next(time.Now())
		id := strings.ToLower(rand.String(32))
		if err := store.GetStore().CreateScheduledInstanceSnapshot(id, c.ClusterID, queued); err != nil {
			logger.Error(err)
			responseBody.Error = "Failed to create first scheduled instance snapshot"
			JSON(w, http.StatusInternalServerError, responseBody)
			return
		}
	}

	responseBody.Success = true
	JSON(w, http.StatusOK, responseBody)
}

type SaveInstanceSnapshotRetentionRequest struct {
	InputValue    string `json:"inputValue"`
	InputTimeUnit string `json:"inputTimeUnit"`
}

func (h *Handler) SaveInstanceSnapshotRetention(w http.ResponseWriter, r *http.Request) {
	responseBody := SaveInstanceSnapshotConfigResponse{}

	// check minimal rbac
	if err := requiresKotsadmVeleroAccess(w, r); err != nil {
		return
	}

	requestBody := SaveInstanceSnapshotRetentionRequest{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		logger.Error(err)
		responseBody.Error = "failed to decode request body"
		JSON(w, http.StatusBadRequest, responseBody)
		return
	}

	clusters, err := store.GetStore().ListClusters()
	if err != nil {
		logger.Error(err)
		responseBody.Error = "Failed to list clusters"
		JSON(w, http.StatusInternalServerError, responseBody)
		return
	}
	if len(clusters) == 0 {
		err := errors.New("No clusters found")
		logger.Error(err)
		responseBody.Error = err.Error()
		JSON(w, http.StatusInternalServerError, responseBody)
		return
	}
	c := clusters[0]

	retention, err := snapshot.FormatTTL(requestBody.InputValue, requestBody.InputTimeUnit)
	if err != nil {
		logger.Error(err)
		responseBody.Error = fmt.Sprintf("Invalid instance snapshot retention: %s %s", requestBody.InputValue, requestBody.InputTimeUnit)
		JSON(w, http.StatusBadRequest, responseBody)
		return
	}

	if c.SnapshotTTL != retention {
		c.SnapshotTTL = retention
		if err := store.GetStore().SetInstanceSnapshotTTL(c.ClusterID, retention); err != nil {
			logger.Error(err)
			responseBody.Error = "Failed to set instance snapshot retention"
			JSON(w, http.StatusInternalServerError, responseBody)
			return
		}
	}

	responseBody.Success = true
	JSON(w, http.StatusOK, responseBody)
}

func requiresKotsadmVeleroAccess(w http.ResponseWriter, r *http.Request) error {
	kotsadmNamespace := util.PodNamespace
	requiresVeleroAccess, err := kotssnapshot.CheckKotsadmVeleroAccess(r.Context(), kotsadmNamespace)
	if err != nil {
		errMsg := "failed to check if kotsadm requires access to velero"
		logger.Error(errors.Wrap(err, errMsg))
		response := types.ErrorResponse{Error: util.StrPointer(errMsg)}
		JSON(w, http.StatusInternalServerError, response)
		return errors.New(errMsg)
	}
	if requiresVeleroAccess {
		errMsg := "kotsadm does not have access to velero"
		response := VeleroRBACResponse{
			Success:                     false,
			Error:                       errMsg,
			KotsadmNamespace:            kotsadmNamespace,
			KotsadmRequiresVeleroAccess: true,
		}
		JSON(w, http.StatusConflict, response)
		return errors.New(errMsg)
	}
	return nil
}
