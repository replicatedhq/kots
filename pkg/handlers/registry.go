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
	dockerregistry "github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/registry"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/version"
)

type UpdateAppRegistryRequest struct {
	Hostname   string `json:"hostname"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Namespace  string `json:"namespace"`
	IsReadOnly bool   `json:"isReadOnly"`
}

type UpdateAppRegistryResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Hostname  string `json:"hostname"`
	Username  string `json:"username"`
	Namespace string `json:"namespace"`
}

type GetAppRegistryResponse struct {
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	Hostname   string `json:"hostname"`
	Namespace  string `json:"namespace"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	IsReadOnly bool   `json:"isReadOnly"`
}

type GetKotsadmRegistryResponse struct {
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	Hostname   string `json:"hostname"`
	Namespace  string `json:"namespace"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	IsReadOnly bool   `json:"isReadOnly"`
}

type ValidateAppRegistryRequest struct {
	Hostname   string `json:"hostname"`
	Namespace  string `json:"namespace"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	IsReadOnly bool   `json:"isReadOnly"`
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
		logger.Error(errors.Wrap(err, "failed to decode UpdateAppRegistry request body"))
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, updateAppRegistryResponse)
		return
	}

	if updateAppRegistryRequest.Namespace == "" {
		err := errors.New("registry namespace is required")
		logger.Error(err)
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, http.StatusBadRequest, updateAppRegistryResponse)
		return
	}

	currentStatus, _, err := store.GetStore().GetTaskStatus("image-rewrite")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get image-rewrite taks status"))
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, updateAppRegistryResponse)
		return
	}

	if currentStatus == "running" {
		err := errors.New("image-rewrite is already running, not starting a new one")
		logger.Error(err)
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, updateAppRegistryResponse)
		return
	}

	if err := store.GetStore().ClearTaskStatus("image-rewrite"); err != nil {
		logger.Error(errors.Wrap(err, "failed to clear image-rewrite taks status"))
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, updateAppRegistryResponse)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app from slug"))
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, updateAppRegistryResponse)
		return
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(foundApp.ID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app registry settings"))
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, updateAppRegistryResponse)
		return
	}

	registryPassword := updateAppRegistryRequest.Password
	if registryPassword == registrytypes.PasswordMask {
		registryPassword = registrySettings.Password
	}

	access := dockerregistry.ActionPush
	if updateAppRegistryRequest.IsReadOnly {
		access = dockerregistry.ActionPull
	}
	err = dockerregistry.CheckAccess(updateAppRegistryRequest.Hostname, updateAppRegistryRequest.Username, registryPassword, updateAppRegistryRequest.Namespace, access)
	if err != nil {
		logger.Infof("Failed to test %s access to %q with user %q: %v", access, updateAppRegistryRequest.Hostname, updateAppRegistryRequest.Username, err)
		JSON(w, 400, NewErrorResponse(err))
		return
	}

	updateAppRegistryResponse.Hostname = updateAppRegistryRequest.Hostname
	updateAppRegistryResponse.Username = updateAppRegistryRequest.Username
	updateAppRegistryResponse.Namespace = updateAppRegistryRequest.Namespace

	if !registrySettingsChanged(updateAppRegistryRequest, registrySettings) {
		updateAppRegistryResponse.Success = true
		JSON(w, http.StatusOK, updateAppRegistryResponse)
		return
	}

	// in a goroutine, start pushing the images to the remote registry
	// we will let this function return while this happens
	go func() {
		skipImagePush := updateAppRegistryRequest.IsReadOnly
		if foundApp.IsAirgap {
			// TODO: pushing images not yet supported in airgapped instances.
			skipImagePush = true
		}

		appDir, err := registry.RewriteImages(
			foundApp.ID, foundApp.CurrentSequence, updateAppRegistryRequest.Hostname,
			updateAppRegistryRequest.Username, registryPassword,
			updateAppRegistryRequest.Namespace, skipImagePush, nil)
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
				logger.Error(errors.Wrap(err, "failed to rewrite images"))
			}
			return
		}
		defer os.RemoveAll(appDir)

		newSequence, err := store.GetStore().CreateAppVersion(foundApp.ID, &foundApp.CurrentSequence, appDir, "Registry Change", false, &version.DownstreamGitOps{})
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to create app version"))
			return
		}

		err = store.GetStore().UpdateRegistry(foundApp.ID, updateAppRegistryRequest.Hostname, updateAppRegistryRequest.Username, updateAppRegistryRequest.Password, updateAppRegistryRequest.Namespace, updateAppRegistryRequest.IsReadOnly)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to update registry"))
			return
		}

		if err := preflight.Run(foundApp.ID, foundApp.Slug, newSequence, foundApp.IsAirgap, appDir); err != nil {
			logger.Error(errors.Wrap(err, "failed to run preflights"))
			return
		}
	}()

	updateAppRegistryResponse.Success = true
	JSON(w, http.StatusOK, updateAppRegistryResponse)
}

func registrySettingsChanged(new UpdateAppRegistryRequest, current registrytypes.RegistrySettings) bool {
	if new.Hostname != current.Hostname {
		return true
	}
	if new.Namespace != current.Namespace {
		return true
	}
	if new.Username != current.Username {
		return true
	}
	if new.Password != registrytypes.PasswordMask && new.Password != current.Password {
		return true
	}
	if new.IsReadOnly != current.IsReadOnly {
		return true
	}
	return false
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

	getAppRegistryResponse.Hostname = settings.Hostname
	getAppRegistryResponse.Namespace = settings.Namespace
	getAppRegistryResponse.Username = settings.Username
	getAppRegistryResponse.IsReadOnly = settings.IsReadOnly

	if settings.Password != "" {
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
	getKotsadmRegistryResponse.IsReadOnly = settings.IsReadOnly
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

		if appSettings.Password != "" {
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

	access := dockerregistry.ActionPush
	if validateAppRegistryRequest.IsReadOnly {
		access = dockerregistry.ActionPull
	}

	err = dockerregistry.CheckAccess(validateAppRegistryRequest.Hostname, validateAppRegistryRequest.Username, password, validateAppRegistryRequest.Namespace, access)
	if err != nil {
		// NOTE: it is possible this is a 500 sometimes
		logger.Infof("Failed to test %s access to %q with user %q: %v", access, validateAppRegistryRequest.Hostname, validateAppRegistryRequest.Username, err)
		JSON(w, 400, NewErrorResponse(err))
		return
	}

	validateAppRegistryResponse.Success = true
	JSON(w, 200, validateAppRegistryResponse)
}
