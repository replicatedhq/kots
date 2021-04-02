package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	snapshot "github.com/replicatedhq/kots/pkg/kotsadmsnapshot"
	snapshottypes "github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	kotssnapshot "github.com/replicatedhq/kots/pkg/snapshot"
	kotssnapshottypes "github.com/replicatedhq/kots/pkg/snapshot/types"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/robfig/cron"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
)

type GlobalSnapshotSettingsResponse struct {
	VeleroVersion   string   `json:"veleroVersion"`
	VeleroPlugins   []string `json:"veleroPlugins"`
	VeleroNamespace string   `json:"veleroNamespace"`
	IsVeleroRunning bool     `json:"isVeleroRunning"`
	ResticVersion   string   `json:"resticVersion"`
	IsResticRunning bool     `json:"isResticRunning"`

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
}

type ConfigureFileSystemSnapshotProviderResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type ConfigureFileSystemSnapshotProviderRequest struct {
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

	kotsadmNamespace := os.Getenv("POD_NAMESPACE")

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

	globalSnapshotSettingsResponse.VeleroVersion = veleroStatus.Version
	globalSnapshotSettingsResponse.VeleroPlugins = veleroStatus.Plugins
	globalSnapshotSettingsResponse.VeleroNamespace = veleroStatus.Namespace
	globalSnapshotSettingsResponse.IsVeleroRunning = veleroStatus.Status == "Ready"
	globalSnapshotSettingsResponse.ResticVersion = veleroStatus.ResticVersion
	globalSnapshotSettingsResponse.IsResticRunning = veleroStatus.ResticStatus == "Ready"
	globalSnapshotSettingsResponse.KotsadmNamespace = kotsadmNamespace
	globalSnapshotSettingsResponse.IsKurl = kurl.IsKurl()
	globalSnapshotSettingsResponse.IsMinimalRBACEnabled = !k8sutil.IsKotsadmClusterScoped(r.Context(), clientset, kotsadmNamespace)

	registryOptions, err := kotsadm.GetKotsadmOptionsFromCluster(kotsadmNamespace, clientset)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to get kotsadm options from cluster"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}

	if updateGlobalSnapshotSettingsRequest.FileSystem != nil {
		// make sure the file system provider is configured and deployed first
		if err := configureFileSystemProvider(r.Context(), clientset, kotsadmNamespace, registryOptions, *updateGlobalSnapshotSettingsRequest.FileSystem); err != nil {
			if _, ok := errors.Cause(err).(*kotssnapshot.ResetFileSystemError); ok {
				globalSnapshotSettingsResponse.Error = err.Error()
				JSON(w, http.StatusConflict, globalSnapshotSettingsResponse)
				return
			}
			logger.Error(err)
			globalSnapshotSettingsResponse.Error = "failed to configure file system provider"
			JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
			return
		}
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
		FileSystem: updateGlobalSnapshotSettingsRequest.FileSystem != nil,

		KotsadmNamespace: kotsadmNamespace,
		RegistryOptions:  &registryOptions,
	}
	updatedStore, err := kotssnapshot.ConfigureStore(r.Context(), options)
	if err != nil {
		if _, ok := errors.Cause(err).(*kotssnapshot.InvalidStoreDataError); ok {
			logger.Error(err)
			globalSnapshotSettingsResponse.Error = "invalid store data"
			JSON(w, http.StatusBadRequest, globalSnapshotSettingsResponse)
			return
		}
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to configure store"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}

	if updatedStore.FileSystem != nil {
		fileSystemConfig, err := kotssnapshot.GetCurrentFileSystemConfig(r.Context(), kotsadmNamespace)
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

	kotsadmNamespace := os.Getenv("POD_NAMESPACE")

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

	globalSnapshotSettingsResponse.VeleroVersion = veleroStatus.Version
	globalSnapshotSettingsResponse.VeleroPlugins = veleroStatus.Plugins
	globalSnapshotSettingsResponse.VeleroNamespace = veleroStatus.Namespace
	globalSnapshotSettingsResponse.IsVeleroRunning = veleroStatus.Status == "Ready"
	globalSnapshotSettingsResponse.ResticVersion = veleroStatus.ResticVersion
	globalSnapshotSettingsResponse.IsResticRunning = veleroStatus.ResticStatus == "Ready"
	globalSnapshotSettingsResponse.KotsadmNamespace = kotsadmNamespace
	globalSnapshotSettingsResponse.IsKurl = kurl.IsKurl()
	globalSnapshotSettingsResponse.IsMinimalRBACEnabled = !k8sutil.IsKotsadmClusterScoped(r.Context(), clientset, kotsadmNamespace)

	store, err := kotssnapshot.GetGlobalStore(r.Context(), kotsadmNamespace, nil)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to get store"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}
	if store == nil {
		err = errors.New("store not found")
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "store not found"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}

	if err := kotssnapshot.Redact(store); err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to redact"
		JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
		return
	}

	if store.FileSystem != nil {
		fileSystemConfig, err := kotssnapshot.GetCurrentFileSystemConfig(r.Context(), kotsadmNamespace)
		if err != nil {
			logger.Error(err)
			globalSnapshotSettingsResponse.Error = "failed to get file system config"
			JSON(w, http.StatusInternalServerError, globalSnapshotSettingsResponse)
			return
		}
		globalSnapshotSettingsResponse.FileSystemConfig = fileSystemConfig
	}

	globalSnapshotSettingsResponse.Store = store
	globalSnapshotSettingsResponse.Success = true

	JSON(w, http.StatusOK, globalSnapshotSettingsResponse)
}

func (h *Handler) ConfigureFileSystemSnapshotProvider(w http.ResponseWriter, r *http.Request) {
	response := ConfigureFileSystemSnapshotProviderResponse{
		Success: false,
	}

	request := ConfigureFileSystemSnapshotProviderRequest{}
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

	namespace := os.Getenv("POD_NAMESPACE")

	registryOptions, err := kotsadm.GetKotsadmOptionsFromCluster(namespace, clientset)
	if err != nil {
		errMsg := "failed to get kotsadm options from cluster"
		response.Error = errMsg
		logger.Error(errors.Wrap(err, errMsg))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: do this asynchronously and use task status to report back

	if err := configureFileSystemProvider(r.Context(), clientset, namespace, registryOptions, request.FileSystemOptions); err != nil {
		if _, ok := errors.Cause(err).(*kotssnapshot.ResetFileSystemError); ok {
			response.Error = err.Error()
			JSON(w, http.StatusConflict, response)
			return
		}
		errMsg := "failed to configure file system provider"
		response.Error = errMsg
		logger.Error(errors.Wrap(err, errMsg))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.Success = true
	response.Namespace = namespace

	JSON(w, http.StatusOK, response)
}

func configureFileSystemProvider(ctx context.Context, clientset kubernetes.Interface, namespace string, registryOptions kotsadmtypes.KotsadmOptions, fileSystemOptions FileSystemOptions) error {
	deployOptions := kotssnapshot.FileSystemDeployOptions{
		Namespace:        namespace,
		IsOpenShift:      k8sutil.IsOpenShift(clientset),
		ForceReset:       fileSystemOptions.ForceReset,
		FileSystemConfig: fileSystemOptions.FileSystemConfig,
	}
	if err := kotssnapshot.DeployFileSystemMinio(ctx, clientset, deployOptions, registryOptions); err != nil {
		return err
	}

	err := k8sutil.WaitForDeploymentReady(ctx, clientset, namespace, kotssnapshot.FileSystemMinioDeploymentName, time.Minute*3)
	if err != nil {
		return errors.Wrap(err, "failed to wait for file system minio")
	}

	err = kotssnapshot.CreateFileSystemMinioBucket(ctx, clientset, namespace, registryOptions)
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

	detectVelero, err := kotssnapshot.DetectVelero(r.Context(), os.Getenv("POD_NAMESPACE"))
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

type SaveSnapshotConfigRequest struct {
	AppID         string `json:"appId"`
	InputValue    string `json:"inputValue"`
	InputTimeUnit string `json:"inputTimeUnit"`
	Schedule      string `json:"schedule"`
	AutoEnabled   bool   `json:"autoEnabled"`
}

type SaveSnapshotConfigResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) SaveSnapshotConfig(w http.ResponseWriter, r *http.Request) {
	responseBody := SaveSnapshotConfigResponse{}

	// check minimal rbac
	if err := requiresKotsadmVeleroAccess(w, r); err != nil {
		return
	}

	requestBody := SaveSnapshotConfigRequest{}
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

type SaveInstanceSnapshotConfigRequest struct {
	InputValue    string `json:"inputValue"`
	InputTimeUnit string `json:"inputTimeUnit"`
	Schedule      string `json:"schedule"`
	AutoEnabled   bool   `json:"autoEnabled"`
}

type SaveInstanceSnapshotConfigResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) SaveInstanceSnapshotConfig(w http.ResponseWriter, r *http.Request) {
	responseBody := SaveInstanceSnapshotConfigResponse{}

	// check minimal rbac
	if err := requiresKotsadmVeleroAccess(w, r); err != nil {
		return
	}

	requestBody := SaveInstanceSnapshotConfigRequest{}
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

func requiresKotsadmVeleroAccess(w http.ResponseWriter, r *http.Request) error {
	kotsadmNamespace := os.Getenv("POD_NAMESPACE")
	requiresVeleroAccess, err := kotssnapshot.CheckKotsadmVeleroAccess(r.Context(), kotsadmNamespace)
	if err != nil {
		errMsg := "failed to check if kotsadm requires access to velero"
		logger.Error(errors.Wrap(err, errMsg))
		response := ErrorResponse{Error: errMsg}
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
