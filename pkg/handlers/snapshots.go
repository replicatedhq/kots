package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	snapshottypes "github.com/replicatedhq/kots/pkg/api/snapshot/types"
	snapshot "github.com/replicatedhq/kots/pkg/kotsadmsnapshot"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/robfig/cron"
	"k8s.io/apimachinery/pkg/util/rand"
)

type GlobalSnapshotSettingsResponse struct {
	VeleroVersion   string   `json:"veleroVersion"`
	VeleroPlugins   []string `json:"veleroPlugins"`
	IsVeleroRunning bool     `json:"isVeleroRunning"`
	ResticVersion   string   `json:"resticVersion"`
	IsResticRunning bool     `json:"isResticRunning"`
	IsKurl          bool     `json:"isKurl"`

	Store   *snapshottypes.Store `json:"store,omitempty"`
	Success bool                 `json:"success"`
	Error   string               `json:"error,omitempty"`
}

type UpdateGlobalSnapshotSettingsRequest struct {
	Provider string `json:"provider"`
	Bucket   string `json:"bucket"`
	Path     string `json:"path"`

	AWS      *snapshottypes.StoreAWS    `json:"aws"`
	Google   *snapshottypes.StoreGoogle `json:"gcp"`
	Azure    *snapshottypes.StoreAzure  `json:"azure"`
	Other    *snapshottypes.StoreOther  `json:"other"`
	Internal bool                       `json:"internal"`
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
		JSON(w, 400, globalSnapshotSettingsResponse)
		return
	}

	veleroStatus, err := snapshot.DetectVelero()
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to detect velero"
		JSON(w, 500, globalSnapshotSettingsResponse)
		return
	}
	if veleroStatus == nil {
		JSON(w, 200, globalSnapshotSettingsResponse)
		return
	}

	globalSnapshotSettingsResponse.VeleroVersion = veleroStatus.Version
	globalSnapshotSettingsResponse.VeleroPlugins = veleroStatus.Plugins
	globalSnapshotSettingsResponse.IsVeleroRunning = veleroStatus.Status == "Ready"
	globalSnapshotSettingsResponse.ResticVersion = veleroStatus.ResticVersion
	globalSnapshotSettingsResponse.IsResticRunning = veleroStatus.ResticStatus == "Ready"
	globalSnapshotSettingsResponse.IsKurl = kurl.IsKurl()

	store, err := snapshot.GetGlobalStore(nil)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to get store"
		JSON(w, 500, globalSnapshotSettingsResponse)
		return
	}

	store.Provider = updateGlobalSnapshotSettingsRequest.Provider
	store.Bucket = updateGlobalSnapshotSettingsRequest.Bucket
	store.Path = updateGlobalSnapshotSettingsRequest.Path

	if updateGlobalSnapshotSettingsRequest.AWS != nil {
		if store.AWS == nil {
			store.AWS = &snapshottypes.StoreAWS{}
		}
		store.Azure = nil
		store.Google = nil
		store.Other = nil
		store.Internal = nil

		store.AWS.UseInstanceRole = updateGlobalSnapshotSettingsRequest.AWS.UseInstanceRole
		if store.AWS.UseInstanceRole {
			store.AWS.AccessKeyID = ""
			store.AWS.SecretAccessKey = ""
		} else {
			if updateGlobalSnapshotSettingsRequest.AWS.AccessKeyID != "" {
				store.AWS.AccessKeyID = updateGlobalSnapshotSettingsRequest.AWS.AccessKeyID
			}
			if updateGlobalSnapshotSettingsRequest.AWS.SecretAccessKey != "" {
				if strings.Contains(updateGlobalSnapshotSettingsRequest.AWS.SecretAccessKey, "REDACTED") {
					logger.Error(err)
					globalSnapshotSettingsResponse.Error = "invalid aws secret access key"
					JSON(w, 400, globalSnapshotSettingsResponse)
					return
				}
				store.AWS.SecretAccessKey = updateGlobalSnapshotSettingsRequest.AWS.SecretAccessKey
			}
			if updateGlobalSnapshotSettingsRequest.AWS.Region != "" {
				store.AWS.Region = updateGlobalSnapshotSettingsRequest.AWS.Region
			}
		}

		if !store.AWS.UseInstanceRole {
			if store.AWS.AccessKeyID == "" || store.AWS.SecretAccessKey == "" || store.AWS.Region == "" {
				globalSnapshotSettingsResponse.Error = "missing access key id and/or secret access key and/or region"
				JSON(w, 400, globalSnapshotSettingsResponse)
				return
			}
		}
	} else if updateGlobalSnapshotSettingsRequest.Google != nil {
		if store.Google == nil {
			store.Google = &snapshottypes.StoreGoogle{}
		}
		store.AWS = nil
		store.Azure = nil
		store.Other = nil
		store.Internal = nil

		store.Google.UseInstanceRole = updateGlobalSnapshotSettingsRequest.Google.UseInstanceRole
		if store.Google.UseInstanceRole {
			store.Google.JSONFile = ""
			if updateGlobalSnapshotSettingsRequest.Google.ServiceAccount != "" {
				store.Google.ServiceAccount = updateGlobalSnapshotSettingsRequest.Google.ServiceAccount
			}
		} else {
			if updateGlobalSnapshotSettingsRequest.Google.JSONFile != "" {
				if strings.Contains(updateGlobalSnapshotSettingsRequest.Google.JSONFile, "REDACTED") {
					logger.Error(err)
					globalSnapshotSettingsResponse.Error = "invalid JSON file"
					JSON(w, 400, globalSnapshotSettingsResponse)
					return
				}
				store.Google.JSONFile = updateGlobalSnapshotSettingsRequest.Google.JSONFile
			}
		}

		if store.Google.UseInstanceRole {
			if store.Google.ServiceAccount == "" {
				globalSnapshotSettingsResponse.Error = "missing service account"
				JSON(w, 400, globalSnapshotSettingsResponse)
				return
			}
		} else {
			if store.Google.JSONFile == "" {
				globalSnapshotSettingsResponse.Error = "missing JSON file"
				JSON(w, 400, globalSnapshotSettingsResponse)
				return
			}
		}

	} else if updateGlobalSnapshotSettingsRequest.Azure != nil {
		if store.Azure == nil {
			store.Azure = &snapshottypes.StoreAzure{}
		}
		store.AWS = nil
		store.Google = nil
		store.Other = nil
		store.Internal = nil

		if updateGlobalSnapshotSettingsRequest.Azure.ResourceGroup != "" {
			store.Azure.ResourceGroup = updateGlobalSnapshotSettingsRequest.Azure.ResourceGroup
		}
		if updateGlobalSnapshotSettingsRequest.Azure.SubscriptionID != "" {
			store.Azure.SubscriptionID = updateGlobalSnapshotSettingsRequest.Azure.SubscriptionID
		}
		if updateGlobalSnapshotSettingsRequest.Azure.TenantID != "" {
			store.Azure.TenantID = updateGlobalSnapshotSettingsRequest.Azure.TenantID
		}
		if updateGlobalSnapshotSettingsRequest.Azure.ClientID != "" {
			store.Azure.ClientID = updateGlobalSnapshotSettingsRequest.Azure.ClientID
		}
		if updateGlobalSnapshotSettingsRequest.Azure.ClientSecret != "" {
			if strings.Contains(updateGlobalSnapshotSettingsRequest.Azure.ClientSecret, "REDACTED") {
				logger.Error(err)
				globalSnapshotSettingsResponse.Error = "invalid client secret"
				JSON(w, 400, globalSnapshotSettingsResponse)
				return
			}
			store.Azure.ClientSecret = updateGlobalSnapshotSettingsRequest.Azure.ClientSecret
		}
		if updateGlobalSnapshotSettingsRequest.Azure.CloudName != "" {
			store.Azure.CloudName = updateGlobalSnapshotSettingsRequest.Azure.CloudName
		}
		if updateGlobalSnapshotSettingsRequest.Azure.StorageAccount != "" {
			store.Azure.StorageAccount = updateGlobalSnapshotSettingsRequest.Azure.StorageAccount
		}

	} else if updateGlobalSnapshotSettingsRequest.Other != nil {
		if store.Other == nil {
			store.Other = &snapshottypes.StoreOther{}
		}
		store.AWS = nil
		store.Google = nil
		store.Azure = nil
		store.Internal = nil

		store.Provider = "aws"
		if updateGlobalSnapshotSettingsRequest.Other.AccessKeyID != "" {
			store.Other.AccessKeyID = updateGlobalSnapshotSettingsRequest.Other.AccessKeyID
		}
		if updateGlobalSnapshotSettingsRequest.Other.SecretAccessKey != "" {
			if strings.Contains(updateGlobalSnapshotSettingsRequest.Other.SecretAccessKey, "REDACTED") {
				logger.Error(err)
				globalSnapshotSettingsResponse.Error = "invalid secret access key"
				JSON(w, 400, globalSnapshotSettingsResponse)
				return
			}
			store.Other.SecretAccessKey = updateGlobalSnapshotSettingsRequest.Other.SecretAccessKey
		}
		if updateGlobalSnapshotSettingsRequest.Other.Region != "" {
			store.Other.Region = updateGlobalSnapshotSettingsRequest.Other.Region
		}
		if updateGlobalSnapshotSettingsRequest.Other.Endpoint != "" {
			store.Other.Endpoint = updateGlobalSnapshotSettingsRequest.Other.Endpoint
		}

		if store.Other.AccessKeyID == "" || store.Other.SecretAccessKey == "" || store.Other.Endpoint == "" || store.Other.Region == "" {
			globalSnapshotSettingsResponse.Error = "access key, secret key, endpoint and region are required"
			JSON(w, 400, globalSnapshotSettingsResponse)
			return
		}
	} else if updateGlobalSnapshotSettingsRequest.Internal {
		if !kurl.IsKurl() {
			globalSnapshotSettingsResponse.Error = "cannot use internal storage on a non-kurl cluster"
			JSON(w, 400, globalSnapshotSettingsResponse)
			return
		}

		if store.Internal == nil {
			store.Internal = &snapshottypes.StoreInternal{}
		}
		store.AWS = nil
		store.Google = nil
		store.Azure = nil
		store.Other = nil

		secret, err := kurl.GetS3Secret()
		if err != nil {
			logger.Error(err)
			globalSnapshotSettingsResponse.Error = err.Error()
			JSON(w, 500, globalSnapshotSettingsResponse)
			return
		}
		if secret == nil {
			logger.Error(errors.New("s3 secret does not exist"))
			globalSnapshotSettingsResponse.Error = "s3 secret does not exist"
			JSON(w, 500, globalSnapshotSettingsResponse)
			return
		}

		store.Provider = "aws"
		store.Bucket = string(secret.Data["velero-local-bucket"])
		store.Path = ""

		store.Internal.AccessKeyID = string(secret.Data["access-key-id"])
		store.Internal.SecretAccessKey = string(secret.Data["secret-access-key"])
		store.Internal.Endpoint = string(secret.Data["endpoint"])
		store.Internal.ObjectStoreClusterIP = string(secret.Data["object-store-cluster-ip"])
		store.Internal.Region = "us-east-1"
	}

	if err := snapshot.ValidateStore(store); err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = errors.Cause(err).Error()
		JSON(w, 400, globalSnapshotSettingsResponse)
		return
	}

	updatedBackupStorageLocation, err := snapshot.UpdateGlobalStore(store)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to update global store"
		JSON(w, 500, globalSnapshotSettingsResponse)
		return
	}

	if err := snapshot.ResetResticRepositories(); err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to try to reset restic repositories"
		JSON(w, 500, globalSnapshotSettingsResponse)
		return
	}

	// most plugins (all?) require that velero be restared after updating
	if err := snapshot.RestartVelero(); err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to try to restart velero"
		JSON(w, 500, globalSnapshotSettingsResponse)
		return
	}

	updatedStore, err := snapshot.GetGlobalStore(updatedBackupStorageLocation)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to update store"
		JSON(w, 500, globalSnapshotSettingsResponse)
		return
	}

	if err := snapshot.Redact(updatedStore); err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to redact"
		JSON(w, 500, globalSnapshotSettingsResponse)
		return
	}

	globalSnapshotSettingsResponse.Store = updatedStore
	globalSnapshotSettingsResponse.Success = true

	JSON(w, 200, globalSnapshotSettingsResponse)
}

func (h *Handler) GetGlobalSnapshotSettings(w http.ResponseWriter, r *http.Request) {
	globalSnapshotSettingsResponse := GlobalSnapshotSettingsResponse{
		Success: false,
	}

	veleroStatus, err := snapshot.DetectVelero()
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to detect velero"
		JSON(w, 500, globalSnapshotSettingsResponse)
		return
	}
	if veleroStatus == nil {
		JSON(w, 200, globalSnapshotSettingsResponse)
		return
	}

	globalSnapshotSettingsResponse.VeleroVersion = veleroStatus.Version
	globalSnapshotSettingsResponse.VeleroPlugins = veleroStatus.Plugins
	globalSnapshotSettingsResponse.IsVeleroRunning = veleroStatus.Status == "Ready"
	globalSnapshotSettingsResponse.ResticVersion = veleroStatus.ResticVersion
	globalSnapshotSettingsResponse.IsResticRunning = veleroStatus.ResticStatus == "Ready"
	globalSnapshotSettingsResponse.IsKurl = kurl.IsKurl()

	store, err := snapshot.GetGlobalStore(nil)
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to get store"
		JSON(w, 500, globalSnapshotSettingsResponse)
		return
	}

	if err := snapshot.Redact(store); err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to redact"
		JSON(w, 500, globalSnapshotSettingsResponse)
		return
	}

	globalSnapshotSettingsResponse.Store = store
	globalSnapshotSettingsResponse.Success = true

	JSON(w, 200, globalSnapshotSettingsResponse)
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

	detectVelero, err := snapshot.DetectVelero()
	if err != nil {
		logger.Error(err)
		getVeleroStatusResponse.IsVeleroInstalled = false
		JSON(w, 500, getVeleroStatusResponse)
		return
	}

	if detectVelero == nil {
		getVeleroStatusResponse.IsVeleroInstalled = false
		JSON(w, 200, getVeleroStatusResponse)
		return
	}

	getVeleroStatusResponse.IsVeleroInstalled = true
	JSON(w, 200, getVeleroStatusResponse)
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
		JSON(w, 200, responseBody)
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
	JSON(w, 200, responseBody)
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
		JSON(w, 200, responseBody)
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
	requiresVeleroAccess, veleroNamespace, err := snapshot.CheckKotsadmVeleroAccess()
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
			KotsadmRequiresVeleroAccess: true,
			VeleroNamespace:             veleroNamespace,
		}
		JSON(w, http.StatusConflict, response)
		return errors.New(errMsg)
	}
	return nil
}
