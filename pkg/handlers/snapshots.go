package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/session"
	"github.com/replicatedhq/kotsadm/pkg/snapshot"
	snapshottypes "github.com/replicatedhq/kotsadm/pkg/snapshot/types"
)

type GlobalSnapshotSettingsResponse struct {
	Store   *snapshottypes.Store `json:"store"`
	Success bool                 `json:"success"`
	Error   string               `json:"error,omitempty"`
}

type UpdateGlobalSnapshotSettingsRequest struct {
	Provider string `json:"provider"`
	Bucket   string `json:"bucket"`
	Path     string `json:"path"`

	AWS    *snapshottypes.StoreAWS    `json:"aws"`
	Google *snapshottypes.StoreGoogle `json:"google"`
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

	store, err := snapshot.GetGlobalStore()
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
				store.AWS.SecretAccessKey = updateGlobalSnapshotSettingsRequest.AWS.SecretAccessKey
			}
		}

		if !store.AWS.UseInstanceRole {
			if store.AWS.AccessKeyID == "" || store.AWS.SecretAccessKey == "" {
				globalSnapshotSettingsResponse.Error = "missing access key id and/or secret access key"
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
		// TODO
	} else if updateGlobalSnapshotSettingsRequest.Other != nil {
		// TODO
	}

	if err := snapshot.UpdateGlobalStore(store); err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to decode request body"
		JSON(w, 400, globalSnapshotSettingsResponse)
		return
	}

	updatedStore, err := snapshot.GetGlobalStore()
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to update store"
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

	store, err := snapshot.GetGlobalStore()
	if err != nil {
		logger.Error(err)
		globalSnapshotSettingsResponse.Error = "failed to get store"
		JSON(w, 500, globalSnapshotSettingsResponse)
		return
	}
	globalSnapshotSettingsResponse.Store = store
	globalSnapshotSettingsResponse.Success = true

	JSON(w, 200, globalSnapshotSettingsResponse)
}
