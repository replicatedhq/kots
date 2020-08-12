package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/kurl"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/snapshot"
	snapshottypes "github.com/replicatedhq/kots/kotsadm/pkg/snapshot/types"
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

func GetSnapshotConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]
	foundApp, err := app.GetFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ttl := &snapshottypes.SnapshotTTL{}
	if foundApp.SnapshotTTL != "" {
		parsedTTL, err := parseTTL(foundApp.SnapshotTTL)
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

	JSON(w, 200, getSnapshotConfigResponse)
}

func parseTTL(s string) (*snapshottypes.ParsedTTL, error) {
	parsedTTLResponse := &snapshottypes.ParsedTTL{}

	ttlMatch, err := regexp.Compile(`^\d+(s|m|h)$`)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid snapshot TTl %v", s)
	}

	matches := FindStringSubmatch(s)
	if len(matches) < 2 {
		return nil, errors.Wrap(err, "failed to get a valid match")
	}

	unit := matches[1]
	quantity := strings.Split(ttlMatch.FindStringSubmatch(s)[0], unit)
	quantityInt, err := strconv.ParseInt(quantity[0], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parseInt quanitity")
	}

	switch unit {
	case "s":
		parsedTTLResponse.Quantity = quantityInt
		parsedTTLResponse.Unit = "seconds"
		break
	case "m":
		parsedTTLResponse.Quantity = quantityInt
		parsedTTLResponse.Unit = "minutes"
		break
	case "h":
		if quantityInt/8766 >= 1 && quantityInt%8766 == 0 {
			parsedTTLResponse.Quantity = quantityInt / 8766
			parsedTTLResponse.Unit = "years"
			break
		}
		if quantityInt/720 >= 1 && quantityInt%720 == 0 {
			parsedTTLResponse.Quantity = quantityInt / 720
			parsedTTLResponse.Unit = "months"
			break
		}
		if quantityInt/168 >= 1 && quantityInt%168 == 0 {
			parsedTTLResponse.Quantity = quantityInt / 168
			parsedTTLResponse.Unit = "weeks"
			break
		}
		if quantityInt/24 >= 1 && quantityInt%24 == 0 {
			parsedTTLResponse.Quantity = quantityInt / 24
			parsedTTLResponse.Unit = "days"
			break
		}
		parsedTTLResponse.Quantity = quantityInt
		parsedTTLResponse.Unit = "hours"
		break
	default:
		return nil, errors.Wrap(nil, "unsupported unit type")
	}
	return parsedTTLResponse, nil
}
