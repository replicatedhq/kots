package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/containers/image/v5/docker"
	"github.com/docker/distribution/registry/api/errcode"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/registry"
	registrytypes "github.com/replicatedhq/kots/kotsadm/pkg/registry/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
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

func (h *Handler) UpdateAppRegistry(w http.ResponseWriter, r *http.Request) {
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

	currentStatus, _, err := store.GetStore().GetTaskStatus("image-rewrite")
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

	if err := store.GetStore().ClearTaskStatus("image-rewrite"); err != nil {
		logger.Error(err)
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, 500, updateAppRegistryResponse)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
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
	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(foundApp.ID)
	if err != nil {
		logger.Error(err)
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, updateAppRegistryResponse)
		return
	}

	if registrySettings != nil {
		if registrySettings.Hostname == updateAppRegistryRequest.Hostname {
			if registrySettings.Namespace == updateAppRegistryRequest.Namespace {

				err := store.GetStore().UpdateRegistry(foundApp.ID, updateAppRegistryRequest.Hostname, updateAppRegistryRequest.Username, updateAppRegistryRequest.Password, updateAppRegistryRequest.Namespace)
				if err != nil {
					logger.Error(err)
					updateAppRegistryResponse.Error = err.Error()
					JSON(w, http.StatusInternalServerError, updateAppRegistryResponse)
					return
				}

				updateAppRegistryResponse.Success = true
				JSON(w, http.StatusOK, updateAppRegistryResponse)
				return
			}
		}
	}

	// in a goroutine, start pushing the images to the remote registry
	// we will let this function return while this happens
	go func() {
		appDir, err := registry.RewriteImages(
			foundApp.ID, foundApp.CurrentSequence, updateAppRegistryRequest.Hostname,
			updateAppRegistryRequest.Username, updateAppRegistryRequest.Password,
			updateAppRegistryRequest.Namespace, nil)
		if err != nil {
			// log credential errors at info level
			causeErr := errors.Cause(err)
			switch causeErr.(type) {
			case docker.ErrUnauthorizedForCredentials, errcode.Errors, errcode.Error, awserr.Error, *url.Error:
				logger.Infof(
					"Failed to rewrite images for host %q and username %q: %v",
					updateAppRegistryRequest.Hostname,
					updateAppRegistryRequest.Username,
					causeErr,
				)
			default:
				logger.Error(err)
			}
			return
		}
		defer os.RemoveAll(appDir)

		newSequence, err := store.GetStore().CreateAppVersion(foundApp.ID, &foundApp.CurrentSequence, appDir, "Registry Change", false, &version.DownstreamGitOps{})
		if err != nil {
			logger.Error(err)
			updateAppRegistryResponse.Error = err.Error()
			JSON(w, http.StatusInternalServerError, updateAppRegistryResponse)
			return
		}

		if err := preflight.Run(foundApp.ID, foundApp.Slug, newSequence, foundApp.IsAirgap, appDir); err != nil {
			logger.Error(err)
			updateAppRegistryResponse.Error = err.Error()
			JSON(w, http.StatusInternalServerError, updateAppRegistryResponse)
			return
		}

		err = store.GetStore().UpdateRegistry(foundApp.ID, updateAppRegistryRequest.Hostname, updateAppRegistryRequest.Username, updateAppRegistryRequest.Password, updateAppRegistryRequest.Namespace)
		if err != nil {
			logger.Error(err)
			return
		}
	}()

	updateAppRegistryResponse.Success = true
	JSON(w, 200, updateAppRegistryResponse)
}

func (h *Handler) GetAppRegistry(w http.ResponseWriter, r *http.Request) {
	getAppRegistryResponse := GetAppRegistryResponse{
		Success: false,
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		getAppRegistryResponse.Error = err.Error()
		JSON(w, 500, getAppRegistryResponse)
		return
	}

	settings, err := store.GetStore().GetRegistryDetailsForApp(foundApp.ID)
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
		getAppRegistryResponse.Password = registrytypes.PasswordMask
	}

	getAppRegistryResponse.Success = true

	JSON(w, 200, getAppRegistryResponse)
}

func (h *Handler) GetKotsadmRegistry(w http.ResponseWriter, r *http.Request) {
	getKotsadmRegistryResponse := GetKotsadmRegistryResponse{
		Success: false,
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
	if settings.Hostname != "" && settings.Username != "" {
		getKotsadmRegistryResponse.Password = registrytypes.PasswordMask
	}

	JSON(w, 200, getKotsadmRegistryResponse)
}

func (h *Handler) ValidateAppRegistry(w http.ResponseWriter, r *http.Request) {
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

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		validateAppRegistryResponse.Error = err.Error()
		JSON(w, 500, validateAppRegistryResponse)
		return
	}

	password := validateAppRegistryRequest.Password
	if password == registrytypes.PasswordMask {
		appSettings, err := store.GetStore().GetRegistryDetailsForApp(foundApp.ID)
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
				JSON(w, 400, NewErrorResponse(err))
				return
			}
			password = kotsadmSettings.Password
		}
	}
	if password == "" || password == registrytypes.PasswordMask {
		err := errors.Errorf("no password found for %s", validateAppRegistryRequest.Hostname)
		JSON(w, 400, NewErrorResponse(err))
		return
	}

	err = dockerregistry.TestPushAccess(validateAppRegistryRequest.Hostname, validateAppRegistryRequest.Username, password, validateAppRegistryRequest.Namespace)
	if err != nil {
		// NOTE: it is possible this is a 500 sometimes
		logger.Infof("Failed to test push access: %v", err)
		JSON(w, 400, NewErrorResponse(err))
		return
	}

	validateAppRegistryResponse.Success = true
	JSON(w, 200, validateAppRegistryResponse)
}
