package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/containers/image/v5/docker"
	"github.com/distribution/distribution/v3/registry/api/errcode"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	dockerregistry "github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/handlers/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight"
	"github.com/replicatedhq/kots/pkg/registry"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/version"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
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

type DockerHubSecretUpdatedResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) DockerHubSecretUpdated(w http.ResponseWriter, r *http.Request) {
	ensureDockerHubSecretResponse := DockerHubSecretUpdatedResponse{
		Success: false,
	}

	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list installed apps"))
		ensureDockerHubSecretResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, ensureDockerHubSecretResponse)
		return
	}

	for _, app := range apps {
		latestSequence, err := store.GetStore().GetLatestAppSequence(app.ID, true)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to get latest app version for app %s", app.Slug))
			logger.Error(errors.Wrap(err, ensureDockerHubSecretResponse.Error))
			JSON(w, http.StatusInternalServerError, ensureDockerHubSecretResponse)
			return
		}

		createNewVersion := true
		isPrimaryVersion := true
		skipPrefligths := false
		deploy := false
		resp, err := updateAppConfig(app, latestSequence, nil, createNewVersion, isPrimaryVersion, skipPrefligths, deploy)
		if err != nil {
			logger.Error(err)
			JSON(w, http.StatusInternalServerError, resp)
			return
		}
	}

	ensureDockerHubSecretResponse.Success = true
	JSON(w, http.StatusOK, ensureDockerHubSecretResponse)
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

	if updateAppRegistryRequest.Hostname == "" {
		if foundApp.IsAirgap {
			JSON(w, http.StatusBadRequest, types.NewErrorResponse(errors.New("registry cannot be removed in airgap installs")))
			return
		}
		// lazy way to clear out all fields
		updateAppRegistryRequest = UpdateAppRegistryRequest{}
	} else {
		err = dockerregistry.CheckAccess(updateAppRegistryRequest.Hostname, updateAppRegistryRequest.Username, registryPassword)
		if err != nil {
			logger.Infof("Failed to test access to %q with user %q: %v", updateAppRegistryRequest.Hostname, updateAppRegistryRequest.Username, err)
			JSON(w, http.StatusBadRequest, types.NewErrorResponse(err))
			return
		}
	}

	updateAppRegistryResponse.Hostname = updateAppRegistryRequest.Hostname
	updateAppRegistryResponse.Username = updateAppRegistryRequest.Username
	updateAppRegistryResponse.Namespace = updateAppRegistryRequest.Namespace

	registryChanged, err := registrySettingsChanged(foundApp, updateAppRegistryRequest, registrySettings)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to check registry settings"))
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, updateAppRegistryResponse)
		return
	}

	if !registryChanged {
		updateAppRegistryResponse.Success = true
		JSON(w, http.StatusOK, updateAppRegistryResponse)
		return
	}

	skipImagePush := updateAppRegistryRequest.IsReadOnly
	if foundApp.IsAirgap {
		// TODO: pushing images not yet supported in airgapped instances.
		skipImagePush = true
	}

	latestSequence, err := store.GetStore().GetLatestAppSequence(foundApp.ID, true)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get latest app sequence for app %s", foundApp.Slug))
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, updateAppRegistryResponse)
		return
	}

	// set task status before starting the goroutine so that the UI can show the status
	if err := store.GetStore().SetTaskStatus("image-rewrite", "Updating registry settings", "running"); err != nil {
		logger.Error(errors.Wrap(err, "failed to set task status"))
		updateAppRegistryResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, updateAppRegistryResponse)
		return
	}

	// in a goroutine, start pushing the images to the remote registry
	// we will let this function return while this happens
	go func() {
		appDir, err := registry.RewriteImages(
			foundApp.ID, latestSequence, updateAppRegistryRequest.Hostname,
			updateAppRegistryRequest.Username, registryPassword,
			updateAppRegistryRequest.Namespace, skipImagePush)
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

		newSequence, err := store.GetStore().CreateAppVersion(foundApp.ID, &latestSequence, appDir, "Registry Change", false, &version.DownstreamGitOps{}, render.Renderer{})
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

func registrySettingsChanged(app *apptypes.App, new UpdateAppRegistryRequest, current registrytypes.RegistrySettings) (bool, error) {
	if new.Hostname != current.Hostname {
		return true, nil
	}
	if new.Namespace != current.Namespace {
		return true, nil
	}
	if new.Username != current.Username {
		return true, nil
	}
	if new.Password != registrytypes.PasswordMask && new.Password != current.Password {
		return true, nil
	}
	if new.IsReadOnly != current.IsReadOnly {
		return true, nil
	}

	// Because an old version can be editted, we may need to push images if registry hostname has changed
	// TODO: Handle namespace changes too
	latestSequence, err := store.GetStore().GetLatestAppSequence(app.ID, true)
	if err != nil {
		return false, errors.Wrap(err, "failed to get latest app sequence")
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm-")
	if err != nil {
		return false, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(app.ID, latestSequence, archiveDir)
	if err != nil {
		return false, errors.Wrap(err, "failed to get version archive")
	}

	secretData, err := os.ReadFile(filepath.Join(archiveDir, "overlays", "midstream", "secret.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			if new.Hostname != "" {
				return true, nil
			} else {
				return false, nil
			}
		}
		return false, errors.Wrap(err, "failed to load image pull secret")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(secretData, nil, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to decode image pull secret")
	}

	if gvk.Group != "" || gvk.Version != "v1" || gvk.Kind != "Secret" {
		return false, errors.Errorf("unexpected secret GVK: %s", gvk.String())
	}

	secret := obj.(*corev1.Secret)
	if secret.Type != "kubernetes.io/dockerconfigjson" {
		return false, errors.Errorf("unexpected secret type: %s", secret.Type)
	}

	dockerConfig := struct {
		Auths map[string]interface{} `json:"auths"`
	}{}

	err = json.Unmarshal(secret.Data[".dockerconfigjson"], &dockerConfig)
	if err != nil {
		return false, errors.Wrap(err, "failed to unmarshal .dockerconfigjson")
	}

	_, ok := dockerConfig.Auths[new.Hostname]
	if !ok {
		// New hostname is not in the auths list, so images have to pushed
		return true, nil
	}

	return false, nil
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
			password = kotsadmSettings.Password
		}
	}

	if password == registrytypes.PasswordMask {
		err := errors.Errorf("no password found for %s", validateAppRegistryRequest.Hostname)
		JSON(w, 400, types.NewErrorResponse(err))
		return
	}

	err = dockerregistry.CheckAccess(validateAppRegistryRequest.Hostname, validateAppRegistryRequest.Username, password)
	if err != nil {
		// NOTE: it is possible this is a 500 sometimes
		logger.Infof("Failed to test access to %q with user %q: %v", validateAppRegistryRequest.Hostname, validateAppRegistryRequest.Username, err)
		JSON(w, 400, types.NewErrorResponse(err))
		return
	}

	validateAppRegistryResponse.Success = true
	JSON(w, 200, validateAppRegistryResponse)
}
