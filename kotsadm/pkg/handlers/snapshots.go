package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/session"
	"github.com/replicatedhq/kotsadm/pkg/snapshot"
	snapshottypes "github.com/replicatedhq/kotsadm/pkg/snapshot/types"
)

type GlobalSnapshotSettingsResponse struct {
	VeleroVersion   string   `json:"veleroVersion"`
	VeleroPlugins   []string `json:"veleroPlugins"`
	IsVeleroRunning bool     `json:"isVeleroRunning"`
	ResticVersion   string   `json:"resticVersion"`
	IsResticRunning bool     `json:"isResticRunning"`

	Store   *snapshottypes.Store `json:"store,omitempty"`
	Success bool                 `json:"success"`
	Error   string               `json:"error,omitempty"`
}

type UpdateGlobalSnapshotSettingsRequest struct {
	Provider string `json:"provider"`
	Bucket   string `json:"bucket"`
	Path     string `json:"path"`

	AWS    *snapshottypes.StoreAWS    `json:"aws"`
	Google *snapshottypes.StoreGoogle `json:"gcp"`
	Azure  *snapshottypes.StoreAzure  `json:"azure"`
	Other  *snapshottypes.StoreOther  `json:"other"`
}

func UpdateGlobalSnapshotSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	globalSnapshotSettingsResponse := GlobalSnapshotSettingsResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to parse authorization header"
		JSON(w, 401, globalSnapshotSettingsResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		globalSnapshotSettingsResponse.Error = "failed to parse authorization header"
		JSON(w, 401, globalSnapshotSettingsResponse)
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

		store.Google.UseInstanceRole = updateGlobalSnapshotSettingsRequest.Google.UseInstanceRole
		if store.Google.UseInstanceRole {
			store.Google.ServiceAccount = ""
		} else {
			if updateGlobalSnapshotSettingsRequest.Google.ServiceAccount != "" {
				if strings.Contains(updateGlobalSnapshotSettingsRequest.Google.ServiceAccount, "REDACTED") {
					logger.Error(err)
					globalSnapshotSettingsResponse.Error = "invalid gcp service account"
					JSON(w, 400, globalSnapshotSettingsResponse)
					return
				}
				store.Google.ServiceAccount = updateGlobalSnapshotSettingsRequest.Google.ServiceAccount
			}
		}

		if !store.Google.UseInstanceRole {
			if store.Google.ServiceAccount == "" {
				globalSnapshotSettingsResponse.Error = "missing service account"
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
		globalSnapshotSettingsResponse.Error = "failed to decode request body"
		JSON(w, 400, globalSnapshotSettingsResponse)
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

func GetGlobalSnapshotSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	globalSnapshotSettingsResponse := GlobalSnapshotSettingsResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to parse authorization header"
		JSON(w, 401, globalSnapshotSettingsResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		globalSnapshotSettingsResponse.Error = "failed to parse authorization header"
		JSON(w, 401, globalSnapshotSettingsResponse)
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
