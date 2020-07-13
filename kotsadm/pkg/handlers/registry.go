package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/registry"
	"github.com/replicatedhq/kots/kotsadm/pkg/task"
	dockerregistry "github.com/replicatedhq/kots/pkg/docker/registry"
)

type UpdateAppRegistryRequest struct {
	Hostname  string `json:"hostname"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Namespace string `json:"namespace"`
}

type UpdateAppRegistryResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Hostname  string `json:"hostname"`
	Username  string `json:"username"`
	Namespace string `json:"namespace"`
}

type GetAppRegistryResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Hostname  string `json:"hostname"`
	Namespace string `json:"namespace"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type GetKotsadmRegistryResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Hostname  string `json:"hostname"`
	Namespace string `json:"namespace"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type ValidateAppRegistryRequest struct {
	Hostname  string `json:"hostname"`
	Namespace string `json:"namespace"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type ValidateAppRegistryResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func UpdateAppRegistry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	updateAppRegistryResponse := UpdateAppRegistryResponse{
		Success: false,
	}

	updateAppRegistryRequest := UpdateAppRegistryRequest{}
	if err := json.NewDecoder(r.Body).Decode(&updateAppRegistryRequest); err != nil {
		logger.Error(err)
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, 500, updateAppRegistryResponse)
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, 401, updateAppRegistryResponse)
		return
	}

	currentStatus, err := task.GetTaskStatus("image-rewrite")
	if err != nil {
		logger.Error(err)
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, 500, updateAppRegistryResponse)
		return
	}

	if currentStatus == "running" {
		err := errors.New("image-rewrite is already running, not starting a new one")
		logger.Error(err)
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, 500, updateAppRegistryResponse)
		return
	}

	if err := task.ClearTaskStatus("image-rewrite"); err != nil {
		logger.Error(err)
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, 500, updateAppRegistryResponse)
		return
	}

	foundApp, err := app.GetFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, 500, updateAppRegistryResponse)
		return
	}

	updateAppRegistryResponse.Hostname = updateAppRegistryRequest.Hostname
	updateAppRegistryResponse.Username = updateAppRegistryRequest.Username
	updateAppRegistryResponse.Namespace = updateAppRegistryRequest.Namespace

	// if hostname and namespace have not changed, we don't need to re-push
	registrySettings, err := registry.GetRegistrySettingsForApp(foundApp.ID)
	if err != nil {
		logger.Error(err)
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, 500, updateAppRegistryResponse)
		return
	}

	if registrySettings != nil {
		if registrySettings.Hostname == updateAppRegistryRequest.Hostname {
			if registrySettings.Namespace == updateAppRegistryRequest.Namespace {

				err := registry.UpdateRegistry(foundApp.ID, updateAppRegistryRequest.Hostname, updateAppRegistryRequest.Username, updateAppRegistryRequest.Password, updateAppRegistryRequest.Namespace)
				if err != nil {
					logger.Error(err)
					updateAppRegistryResponse.Error = err.Error()
					JSON(w, 500, updateAppRegistryResponse)
					return
				}

				updateAppRegistryResponse.Success = true
				JSON(w, 200, updateAppRegistryResponse)
				return
			}
		}
	}

	// in a goroutine, start pushing the images to the remote registry
	// we will let this function return while this happens
	go func() {
		if err := registry.RewriteImages(foundApp.ID, foundApp.CurrentSequence, updateAppRegistryRequest.Hostname, updateAppRegistryRequest.Username, updateAppRegistryRequest.Password,
			updateAppRegistryRequest.Namespace, nil); err != nil {
			logger.Error(err)
			return
		}

		err = registry.UpdateRegistry(foundApp.ID, updateAppRegistryRequest.Hostname, updateAppRegistryRequest.Username, updateAppRegistryRequest.Password, updateAppRegistryRequest.Namespace)
		if err != nil {
			logger.Error(err)
			return
		}
	}()

	updateAppRegistryResponse.Success = true
	JSON(w, 200, updateAppRegistryResponse)
}

func GetAppRegistry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	getAppRegistryResponse := GetAppRegistryResponse{
		Success: false,
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		getAppRegistryResponse.Error = err.Error()
		JSON(w, 401, getAppRegistryResponse)
		return
	}

	foundApp, err := app.GetFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		getAppRegistryResponse.Error = err.Error()
		JSON(w, 500, getAppRegistryResponse)
		return
	}

	settings, err := registry.GetRegistrySettingsForApp(foundApp.ID)
	if err != nil {
		logger.Error(err)
		getAppRegistryResponse.Error = err.Error()
		JSON(w, 500, getAppRegistryResponse)
		return
	}

	if settings != nil {
		getAppRegistryResponse.Hostname = settings.Hostname
		getAppRegistryResponse.Namespace = settings.Namespace
		getAppRegistryResponse.Username = settings.Username
		getAppRegistryResponse.Password = registry.PasswordMask
	}

	getAppRegistryResponse.Success = true

	JSON(w, 200, getAppRegistryResponse)
}

func GetKotsadmRegistry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	getKotsadmRegistryResponse := GetKotsadmRegistryResponse{
		Success: false,
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		getKotsadmRegistryResponse.Error = err.Error()
		JSON(w, 401, getKotsadmRegistryResponse)
		return
	}

	settings, err := registry.GetKotsadmRegistry()
	if err != nil {
		logger.Error(err)
		getKotsadmRegistryResponse.Error = err.Error()
		JSON(w, 500, getKotsadmRegistryResponse)
		return
	}

	getKotsadmRegistryResponse.Success = true
	getKotsadmRegistryResponse.Hostname = settings.Hostname
	getKotsadmRegistryResponse.Namespace = settings.Namespace
	getKotsadmRegistryResponse.Username = settings.Username
	getKotsadmRegistryResponse.Password = registry.PasswordMask

	JSON(w, 200, getKotsadmRegistryResponse)
}

func ValidateAppRegistry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	validateAppRegistryResponse := ValidateAppRegistryResponse{
		Success: false,
	}

	validateAppRegistryRequest := ValidateAppRegistryRequest{}
	if err := json.NewDecoder(r.Body).Decode(&validateAppRegistryRequest); err != nil {
		logger.Error(err)
		validateAppRegistryResponse.Error = err.Error()
		JSON(w, 500, validateAppRegistryResponse)
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		validateAppRegistryResponse.Error = err.Error()
		JSON(w, 401, validateAppRegistryResponse)
		return
	}

	foundApp, err := app.GetFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		validateAppRegistryResponse.Error = err.Error()
		JSON(w, 500, validateAppRegistryResponse)
		return
	}

	password := validateAppRegistryRequest.Password
	if password == registry.PasswordMask {
		appSettings, err := registry.GetRegistrySettingsForApp(foundApp.ID)
		if err != nil {
			logger.Error(err)
			validateAppRegistryResponse.Error = err.Error()
			JSON(w, 500, validateAppRegistryResponse)
			return
		}

		if appSettings != nil && appSettings.Password != "" {
			password = appSettings.Password

		} else {
			kotsadmSettings, err := registry.GetKotsadmRegistry()
			if err != nil {
				logger.Error(err)
				validateAppRegistryResponse.Error = err.Error()
				JSON(w, 500, validateAppRegistryResponse)
				return
			}

			if kotsadmSettings.Hostname != validateAppRegistryRequest.Hostname || kotsadmSettings.Password == "" {
				err := errors.Errorf("no password found for %s", validateAppRegistryRequest.Hostname)
				logger.Error(err)
				validateAppRegistryResponse.Error = err.Error()
				JSON(w, 400, validateAppRegistryResponse)
				return
			}
			password = kotsadmSettings.Password
		}
	}
	if password == "" || password == registry.PasswordMask {
		err := errors.Errorf("no password found for %s", validateAppRegistryRequest.Hostname)
		logger.Error(err)
		validateAppRegistryResponse.Error = err.Error()
		JSON(w, 400, validateAppRegistryResponse)
		return
	}

	err = dockerregistry.TestPushAccess(validateAppRegistryRequest.Hostname, validateAppRegistryRequest.Username, password, validateAppRegistryRequest.Namespace)
	if err != nil {
		logger.Error(err)
		validateAppRegistryResponse.Error = err.Error()
		JSON(w, 500, validateAppRegistryResponse)
		return
	}

	validateAppRegistryResponse.Success = true
	JSON(w, 200, validateAppRegistryResponse)
}
