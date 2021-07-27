package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/config"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/crypto"
	kotsadmconfig "github.com/replicatedhq/kots/pkg/kotsadmconfig"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/preflight"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

type UpdateAppConfigRequest struct {
	Sequence         int64                     `json:"sequence"`
	CreateNewVersion bool                      `json:"createNewVersion"`
	ConfigGroups     []kotsv1beta1.ConfigGroup `json:"configGroups"`
}

type LiveAppConfigRequest struct {
	Sequence     int64                     `json:"sequence"`
	ConfigGroups []kotsv1beta1.ConfigGroup `json:"configGroups"`
}

type UpdateAppConfigResponse struct {
	Success       bool     `json:"success"`
	Error         string   `json:"error,omitempty"`
	RequiredItems []string `json:"requiredItems,omitempty"`
}

type LiveAppConfigResponse struct {
	Success      bool                      `json:"success"`
	Error        string                    `json:"error,omitempty"`
	ConfigGroups []kotsv1beta1.ConfigGroup `json:"configGroups"`
}

type CurrentAppConfigResponse struct {
	Success      bool                      `json:"success"`
	Error        string                    `json:"error,omitempty"`
	ConfigGroups []kotsv1beta1.ConfigGroup `json:"configGroups"`
}

func (h *Handler) UpdateAppConfig(w http.ResponseWriter, r *http.Request) {
	updateAppConfigResponse := UpdateAppConfigResponse{
		Success: false,
	}

	updateAppConfigRequest := UpdateAppConfigRequest{}
	if err := json.NewDecoder(r.Body).Decode(&updateAppConfigRequest); err != nil {
		logger.Error(err)
		updateAppConfigResponse.Error = "failed to decode request body"
		JSON(w, http.StatusBadRequest, updateAppConfigResponse)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		updateAppConfigResponse.Error = "failed to get app from app slug"
		JSON(w, http.StatusInternalServerError, updateAppConfigResponse)
		return
	}

	isEditbale, err := isVersionConfigEditable(foundApp, updateAppConfigRequest.Sequence)
	if err != nil {
		updateAppConfigResponse.Error = "failed to check if version is editable"
		logger.Error(errors.Wrap(err, updateAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, updateAppConfigResponse)
		return
	}

	if !isEditbale {
		updateAppConfigResponse.Error = "this version cannot be edited"
		logger.Error(errors.Wrap(err, updateAppConfigResponse.Error))
		JSON(w, http.StatusForbidden, updateAppConfigResponse)
		return
	}

	createNewVersion, err := shouldCreateNewAppVersion(foundApp.ID, updateAppConfigRequest.Sequence)
	if err != nil {
		updateAppConfigResponse.Error = "failed to check if version should be created"
		logger.Error(errors.Wrap(err, updateAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, updateAppConfigResponse)
		return
	}

	isPrimaryVersion := true
	skipPrefligths := false
	deploy := false
	resp, err := updateAppConfig(foundApp, updateAppConfigRequest.Sequence, updateAppConfigRequest.ConfigGroups, createNewVersion, isPrimaryVersion, skipPrefligths, deploy)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, resp)
		return
	}

	if len(resp.RequiredItems) > 0 {
		JSON(w, http.StatusBadRequest, resp)
		return
	}

	JSON(w, http.StatusOK, UpdateAppConfigResponse{Success: true})
}

func (h *Handler) LiveAppConfig(w http.ResponseWriter, r *http.Request) {
	liveAppConfigResponse := LiveAppConfigResponse{
		Success: false,
	}

	liveAppConfigRequest := LiveAppConfigRequest{}
	if err := json.NewDecoder(r.Body).Decode(&liveAppConfigRequest); err != nil {
		logger.Error(err)
		liveAppConfigResponse.Error = "failed to decode request body"
		JSON(w, http.StatusBadRequest, liveAppConfigResponse)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		liveAppConfigResponse.Error = "failed to get app from app slug"
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	appLicense, err := store.GetStore().GetLatestLicenseForApp(foundApp.ID)
	if err != nil {
		logger.Error(err)
		liveAppConfigResponse.Error = "failed to get license for app"
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		logger.Error(err)
		liveAppConfigResponse.Error = "failed to create temp dir"
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(foundApp.ID, liveAppConfigRequest.Sequence, archiveDir)
	if err != nil {
		logger.Error(err)
		liveAppConfigResponse.Error = "failed to get app version archive"
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		logger.Error(err)
		liveAppConfigResponse.Error = "failed to load kots kinds from path"
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	// get values from request
	configValues := map[string]template.ItemValue{}

	for _, group := range liveAppConfigRequest.ConfigGroups {
		for _, item := range group.Items {
			// collect all repeatable items
			// Future Note:  This could be refactored to use CountByGroup as the control.  Front end provides the exact CountByGroup it wants, back end takes care of ValuesByGroup entries.
			// this way the front end doesn't have to add anything to ValuesByGroup, it just sets values there.
			if item.Repeatable {
				for valuesByGroupName, groupValues := range item.ValuesByGroup {
					config.CreateVariadicValues(&item, valuesByGroupName)

					for fieldName, subItem := range groupValues {
						itemValue := template.ItemValue{
							Value:          subItem,
							RepeatableItem: item.Name,
						}
						if item.Filename != "" {
							itemValue.Filename = fieldName
						}
						configValues[fieldName] = itemValue
					}
				}
				continue
			}

			generatedValue := template.ItemValue{}
			if item.Value.Type == multitype.String {
				generatedValue.Value = item.Value.StrVal
			} else {
				generatedValue.Value = item.Value.BoolVal
			}
			if item.Default.Type == multitype.String {
				generatedValue.Default = item.Default.StrVal
			} else {
				generatedValue.Default = item.Default.BoolVal
			}
			if item.Type == "file" {
				generatedValue.Filename = item.Filename
			}
			configValues[item.Name] = generatedValue
		}
	}

	registryInfo, err := store.GetStore().GetRegistryDetailsForApp(foundApp.ID)
	if err != nil {
		logger.Error(err)
		liveAppConfigResponse.Error = "failed to get app registry info"
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	localRegistry := template.LocalRegistry{
		Host:      registryInfo.Hostname,
		Namespace: registryInfo.Namespace,
		Username:  registryInfo.Username,
		Password:  registryInfo.Password,
		ReadOnly:  registryInfo.IsReadOnly,
	}

	versionInfo := template.VersionInfoFromInstallation(liveAppConfigRequest.Sequence+1, foundApp.IsAirgap, kotsKinds.Installation.Spec) // sequence +1 because the sequence will be incremented on save (and we want the preview to be accurate)
	renderedConfig, err := kotsconfig.TemplateConfigObjects(kotsKinds.Config, configValues, appLicense, localRegistry, &versionInfo, kotsKinds.IdentityConfig, os.Getenv("POD_NAMESPACE"))
	if err != nil {
		logger.Error(err)
		liveAppConfigResponse.Error = "failed to render templates"
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	configGroups := []kotsv1beta1.ConfigGroup{}
	if renderedConfig != nil {
		configGroups = renderedConfig.Spec.Groups
	}

	JSON(w, http.StatusOK, LiveAppConfigResponse{Success: true, ConfigGroups: configGroups})
}

func (h *Handler) CurrentAppConfig(w http.ResponseWriter, r *http.Request) {
	currentAppConfigResponse := CurrentAppConfigResponse{
		Success: false,
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		currentAppConfigResponse.Error = "failed to get app from app slug"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	appLicense, err := store.GetStore().GetLatestLicenseForApp(foundApp.ID)
	if err != nil {
		logger.Error(err)
		currentAppConfigResponse.Error = "failed to get license for app"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		logger.Error(err)
		currentAppConfigResponse.Error = "failed to parse app sequence"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		logger.Error(err)
		currentAppConfigResponse.Error = "failed to create temp dir"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(foundApp.ID, int64(sequence), archiveDir)
	if err != nil {
		logger.Error(err)
		currentAppConfigResponse.Error = "failed to get app version archive"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		logger.Error(err)
		currentAppConfigResponse.Error = "failed to load kots kinds from path"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	// get values from saved app version
	configValues := map[string]template.ItemValue{}

	for key, value := range kotsKinds.ConfigValues.Spec.Values {
		generatedValue := template.ItemValue{
			Default:        value.Default,
			Value:          value.Value,
			Filename:       value.Filename,
			RepeatableItem: value.RepeatableItem,
		}
		configValues[key] = generatedValue
	}

	registryInfo, err := store.GetStore().GetRegistryDetailsForApp(foundApp.ID)
	if err != nil {
		logger.Error(err)
		currentAppConfigResponse.Error = "failed to get app registry info"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	localRegistry := template.LocalRegistry{
		Host:      registryInfo.Hostname,
		Namespace: registryInfo.Namespace,
		Username:  registryInfo.Username,
		Password:  registryInfo.Password,
		ReadOnly:  registryInfo.IsReadOnly,
	}

	versionInfo := template.VersionInfoFromInstallation(int64(sequence)+1, foundApp.IsAirgap, kotsKinds.Installation.Spec) // sequence +1 because the sequence will be incremented on save (and we want the preview to be accurate)
	renderedConfig, err := kotsconfig.TemplateConfigObjects(kotsKinds.Config, configValues, appLicense, localRegistry, &versionInfo, kotsKinds.IdentityConfig, os.Getenv("POD_NAMESPACE"))
	if err != nil {
		logger.Error(err)
		currentAppConfigResponse.Error = "failed to render templates"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	configGroups := []kotsv1beta1.ConfigGroup{}
	if renderedConfig != nil {
		configGroups = renderedConfig.Spec.Groups
	}

	JSON(w, http.StatusOK, CurrentAppConfigResponse{Success: true, ConfigGroups: configGroups})
}

func isVersionConfigEditable(app *apptypes.App, sequence int64) (bool, error) {
	// Only latest and currently deployed versions can be edited
	if app.CurrentSequence == sequence {
		return true, nil
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(app.ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get downstreams")
	}

	for _, d := range downstreams {
		parentSequence, err := store.GetStore().GetCurrentParentSequence(app.ID, d.ClusterID)
		if err != nil {
			return false, errors.Wrap(err, "failed to get downstream parent sequence")
		}

		if parentSequence == sequence {
			return true, nil
		}
	}

	return false, nil
}

func shouldCreateNewAppVersion(appID string, sequence int64) (bool, error) {
	// Updates are allowed only for sequence 0 and only when it's pending config.
	if sequence > 0 {
		return true, nil
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(appID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get downstreams")
	}

	for _, d := range downstreams {
		status, err := store.GetStore().GetStatusForVersion(appID, d.ClusterID, sequence)
		if err != nil {
			return false, errors.Wrap(err, "failed to get version status")
		}
		if status == storetypes.VersionPendingConfig {
			return false, nil
		}
	}

	return true, nil
}

// if isPrimaryVersion is false, missing a required config field will not cause a failure, and instead will create
// the app version with status needs_config
func updateAppConfig(updateApp *apptypes.App, sequence int64, configGroups []kotsv1beta1.ConfigGroup, createNewVersion bool, isPrimaryVersion bool, skipPreflights bool, deploy bool) (UpdateAppConfigResponse, error) {
	updateAppConfigResponse := UpdateAppConfigResponse{
		Success: false,
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		updateAppConfigResponse.Error = "failed to create temp dir"
		return updateAppConfigResponse, err
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(updateApp.ID, sequence, archiveDir)
	if err != nil {
		updateAppConfigResponse.Error = "failed to get app version archive"
		return updateAppConfigResponse, err
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		updateAppConfigResponse.Error = "failed to load kots kinds from path"
		return updateAppConfigResponse, err
	}

	// check for unset required items
	requiredItems := make([]string, 0, 0)
	requiredItemsTitles := make([]string, 0, 0)
	for _, group := range configGroups {
		if group.When == "false" {
			continue
		}
		for _, item := range group.Items {
			if kotsadmconfig.IsRequiredItem(item) && kotsadmconfig.IsUnsetItem(item) {
				requiredItems = append(requiredItems, item.Name)
				if item.Title != "" {
					requiredItemsTitles = append(requiredItemsTitles, item.Title)
				} else {
					requiredItemsTitles = append(requiredItemsTitles, item.Name)
				}
			}
		}
	}

	// not having all the required items is only a failure for the version that the user intended to edit
	if len(requiredItems) > 0 && isPrimaryVersion {
		updateAppConfigResponse.RequiredItems = requiredItems
		updateAppConfigResponse.Error = fmt.Sprintf("The following fields are required: %s", strings.Join(requiredItemsTitles, ", "))
		return updateAppConfigResponse, nil
	}

	// we don't merge, this is a wholesale replacement of the config values
	// so we don't need the complex logic in kots, we can just write
	values := kotsKinds.ConfigValues.Spec.Values
	updatedValues, err := updateAppConfigValues(values, configGroups, kotsKinds.Installation.Spec.EncryptionKey)

	if kotsKinds.ConfigValues == nil {
		updateAppConfigResponse.Error = "no config values found"
		return updateAppConfigResponse, errors.New("no config values found")
	}

	kotsKinds.ConfigValues.Spec.Values = updatedValues

	configValuesSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
	if err != nil {
		updateAppConfigResponse.Error = "failed to marshal config values spec"
		return updateAppConfigResponse, err
	}

	if err := ioutil.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"), []byte(configValuesSpec), 0644); err != nil {
		updateAppConfigResponse.Error = "failed to write config.yaml to upstream/userdata"
		return updateAppConfigResponse, err
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(updateApp.ID)
	if err != nil {
		updateAppConfigResponse.Error = "failed to get registry settings"
		return updateAppConfigResponse, err
	}

	app, err := store.GetStore().GetApp(updateApp.ID)
	if err != nil {
		updateAppConfigResponse.Error = "failed to get app"
		return updateAppConfigResponse, err
	}
	downstreams, err := store.GetStore().ListDownstreamsForApp(updateApp.ID)
	if err != nil {
		updateAppConfigResponse.Error = "failed to list downstreams for app"
		return updateAppConfigResponse, err
	}

	if app.CurrentSequence != sequence {
		// We are modifying an old version, registry settings may not match what the user has set
		// for the app.  Midstream in version archive is the only place we can get them from.
		versionRegistrySettings, err := midstream.LoadPrivateRegistryInfo(archiveDir)
		if err != nil {
			updateAppConfigResponse.Error = "failed to get version registry settings"
			return updateAppConfigResponse, err
		}

		if versionRegistrySettings == nil {
			registrySettings = registrytypes.RegistrySettings{}
		} else {
			// TODO: missing namespace
			registrySettings.Hostname = versionRegistrySettings.Hostname
			registrySettings.Username = versionRegistrySettings.Username
			registrySettings.Password = versionRegistrySettings.Password
		}
	}

	err = render.RenderDir(archiveDir, app, downstreams, registrySettings, createNewVersion)
	if err != nil {
		updateAppConfigResponse.Error = "failed to render archive directory"
		return updateAppConfigResponse, err
	}

	if createNewVersion {
		newSequence, err := store.GetStore().CreateAppVersion(updateApp.ID, &updateApp.CurrentSequence, archiveDir, "Config Change", false, &version.DownstreamGitOps{})
		if err != nil {
			updateAppConfigResponse.Error = "failed to create an app version"
			return updateAppConfigResponse, err
		}
		sequence = newSequence
	} else {
		if err := kotsadmconfig.UpdateConfigValuesInDB(archiveDir, updateApp.ID, int64(sequence)); err != nil {
			updateAppConfigResponse.Error = "failed to update config values in db"
			return updateAppConfigResponse, err
		}

		if err := store.GetStore().CreateAppVersionArchive(updateApp.ID, int64(sequence), archiveDir); err != nil {
			updateAppConfigResponse.Error = "failed to create app version archive"
			return updateAppConfigResponse, err
		}
	}

	if err := store.GetStore().SetDownstreamVersionPendingPreflight(updateApp.ID, int64(sequence)); err != nil {
		updateAppConfigResponse.Error = "failed to set downstream status to 'pending preflight'"
		return updateAppConfigResponse, err
	}

	if !skipPreflights {
		if err := preflight.Run(updateApp.ID, updateApp.Slug, int64(sequence), updateApp.IsAirgap, archiveDir); err != nil {
			updateAppConfigResponse.Error = errors.Cause(err).Error()
			return updateAppConfigResponse, err
		}
	}

	if deploy {
		err := version.DeployVersion(updateApp.ID, sequence)
		if err != nil {
			updateAppConfigResponse.Error = "failed to deploy"
			return updateAppConfigResponse, err
		}
	}

	updateAppConfigResponse.Success = true
	return updateAppConfigResponse, nil
}

func updateAppConfigValues(values map[string]kotsv1beta1.ConfigValue, configGroups []kotsv1beta1.ConfigGroup, encryptionKey string) (map[string]kotsv1beta1.ConfigValue, error) {
	for _, group := range configGroups {
		for _, item := range group.Items {
			if item.Type == "file" {
				v := values[item.Name]
				v.Filename = item.Filename
				values[item.Name] = v
			}
			if item.Value.Type == multitype.Bool {
				updatedValue := item.Value.BoolVal
				v := values[item.Name]
				v.Value = strconv.FormatBool(updatedValue)
				values[item.Name] = v
			} else if item.Value.Type == multitype.String {
				updatedValue := item.Value.String()
				if item.Type == "password" {
					// encrypt using the key
					cipher, err := crypto.AESCipherFromString(encryptionKey)
					if err != nil {
						return nil, errors.Wrap(err, "failed to load encryption cipher")
					}

					// if the decryption succeeds, don't encrypt again
					_, err = decrypt(updatedValue, cipher)
					if err != nil {
						updatedValue = base64.StdEncoding.EncodeToString(cipher.Encrypt([]byte(updatedValue)))
					}
				}

				v := values[item.Name]
				v.Value = updatedValue
				values[item.Name] = v
			}
			for _, repeatableValues := range item.ValuesByGroup {
				// clear out all variadic values for this group first
				for name, value := range values {
					if value.RepeatableItem == item.Name {
						delete(values, name)
					}
				}
				// add variadic groups back in declaratively
				for itemName, valueItem := range repeatableValues {
					v := values[itemName]
					v.Value = fmt.Sprintf("%v", valueItem)
					v.RepeatableItem = item.Name
					values[itemName] = v
				}
			}
		}
	}
	return values, nil
}
func decrypt(input string, cipher *crypto.AESCipher) (string, error) {
	if cipher == nil {
		return "", errors.New("cipher not defined")
	}

	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", errors.Wrap(err, "failed to base64 decode")
	}

	decrypted, err := cipher.Decrypt(decoded)
	if err != nil {
		return "", errors.Wrap(err, "failed to decrypt")
	}

	return string(decrypted), nil
}

type SetAppConfigValuesRequest struct {
	ConfigValues   []byte `json:"configValues"`
	Merge          bool   `json:"merge"`
	Deploy         bool   `json:"deploy"`
	SkipPreflights bool   `json:"skipPreflights"`
}

type SetAppConfigValuesResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) SetAppConfigValues(w http.ResponseWriter, r *http.Request) {
	setAppConfigValuesResponse := SetAppConfigValuesResponse{
		Success: false,
	}

	setAppConfigValuesRequest := SetAppConfigValuesRequest{}
	if err := json.NewDecoder(r.Body).Decode(&setAppConfigValuesRequest); err != nil {
		setAppConfigValuesResponse.Error = "failed to decode request body"
		logger.Error(errors.Wrap(err, setAppConfigValuesResponse.Error))
		JSON(w, http.StatusBadRequest, setAppConfigValuesResponse)
		return
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, gvk, err := decode(setAppConfigValuesRequest.ConfigValues, nil, nil)
	if err != nil {
		setAppConfigValuesResponse.Error = "failed to decode config values"
		logger.Error(errors.Wrap(err, setAppConfigValuesResponse.Error))
		JSON(w, http.StatusBadRequest, setAppConfigValuesResponse)
		return
	}

	if gvk.String() != "kots.io/v1beta1, Kind=ConfigValues" {
		setAppConfigValuesResponse.Error = fmt.Sprintf("%q is not a valid ConfigValues GVK", gvk.String())
		logger.Errorf(setAppConfigValuesResponse.Error)
		JSON(w, http.StatusInternalServerError, setAppConfigValuesResponse)
		return
	}
	newConfigValues := decoded.(*kotsv1beta1.ConfigValues)

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		setAppConfigValuesResponse.Error = "failed to get app from app slug"
		logger.Error(errors.Wrap(err, setAppConfigValuesResponse.Error))
		JSON(w, http.StatusInternalServerError, setAppConfigValuesResponse)
		return
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		setAppConfigValuesResponse.Error = "failed to create temp dir"
		logger.Error(errors.Wrap(err, setAppConfigValuesResponse.Error))
		JSON(w, http.StatusInternalServerError, setAppConfigValuesResponse)
		return
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(foundApp.ID, foundApp.CurrentSequence, archiveDir)
	if err != nil {
		setAppConfigValuesResponse.Error = "failed to get app version archive"
		logger.Error(errors.Wrap(err, setAppConfigValuesResponse.Error))
		JSON(w, http.StatusInternalServerError, setAppConfigValuesResponse)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		setAppConfigValuesResponse.Error = "failed to load kots kinds from path"
		logger.Error(errors.Wrap(err, setAppConfigValuesResponse.Error))
		JSON(w, http.StatusInternalServerError, setAppConfigValuesResponse)
		return
	}

	if kotsKinds.Config == nil {
		setAppConfigValuesResponse.Error = fmt.Sprintf("app %s does not have a config", foundApp.Slug)
		logger.Errorf(setAppConfigValuesResponse.Error)
		JSON(w, http.StatusInternalServerError, setAppConfigValuesResponse)
		return
	}

	if setAppConfigValuesRequest.Merge {
		if err := kotsKinds.DecryptConfigValues(); err != nil {
			setAppConfigValuesResponse.Error = "failed to decrypt existing values"
			logger.Error(errors.Wrap(err, setAppConfigValuesResponse.Error))
			JSON(w, http.StatusInternalServerError, setAppConfigValuesResponse)
			return
		}

		newConfigValues, err = mergeConfigValues(kotsKinds.Config, kotsKinds.ConfigValues, newConfigValues)
		if err != nil {
			setAppConfigValuesResponse.Error = "failed to create new config"
			logger.Error(errors.Wrap(err, setAppConfigValuesResponse.Error))
			JSON(w, http.StatusInternalServerError, setAppConfigValuesResponse)
			return
		}
	}

	newConfig, err := updateConfigObject(kotsKinds.Config, newConfigValues, setAppConfigValuesRequest.Merge)
	if err != nil {
		setAppConfigValuesResponse.Error = "failed to create new config object"
		logger.Error(errors.Wrap(err, setAppConfigValuesResponse.Error))
		JSON(w, http.StatusInternalServerError, setAppConfigValuesResponse)
		return
	}

	configValueMap := map[string]template.ItemValue{}
	for key, value := range newConfigValues.Spec.Values {
		generatedValue := template.ItemValue{
			Default:        value.Default,
			Value:          value.Value,
			RepeatableItem: value.RepeatableItem,
		}
		configValueMap[key] = generatedValue
	}

	registryInfo, err := store.GetStore().GetRegistryDetailsForApp(foundApp.ID)
	if err != nil {
		setAppConfigValuesResponse.Error = "failed to get app registry info"
		logger.Error(errors.Wrap(err, setAppConfigValuesResponse.Error))
		JSON(w, http.StatusInternalServerError, setAppConfigValuesResponse)
		return
	}

	localRegistry := template.LocalRegistry{
		Host:      registryInfo.Hostname,
		Namespace: registryInfo.Namespace,
		Username:  registryInfo.Username,
		Password:  registryInfo.Password,
		ReadOnly:  registryInfo.IsReadOnly,
	}

	versionInfo := template.VersionInfoFromInstallation(foundApp.CurrentSequence+1, foundApp.IsAirgap, kotsKinds.Installation.Spec) // sequence +1 because the sequence will be incremented on save (and we want the preview to be accurate)
	renderedConfig, err := kotsconfig.TemplateConfigObjects(newConfig, configValueMap, kotsKinds.License, localRegistry, &versionInfo, kotsKinds.IdentityConfig, os.Getenv("POD_NAMESPACE"))
	if err != nil {
		setAppConfigValuesResponse.Error = "failed to render templates"
		logger.Error(errors.Wrap(err, setAppConfigValuesResponse.Error))
		JSON(w, http.StatusInternalServerError, setAppConfigValuesResponse)
		return
	}

	if renderedConfig == nil {
		setAppConfigValuesResponse.Error = "application does not have config"
		logger.Error(errors.New(setAppConfigValuesResponse.Error))
		JSON(w, http.StatusBadRequest, setAppConfigValuesResponse)
		return
	}

	createNewVersion := true
	isPrimaryVersion := true // see comment in updateAppConfig
	resp, err := updateAppConfig(foundApp, foundApp.CurrentSequence, renderedConfig.Spec.Groups, createNewVersion, isPrimaryVersion, setAppConfigValuesRequest.SkipPreflights, setAppConfigValuesRequest.Deploy)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create new version"))
		JSON(w, http.StatusInternalServerError, resp)
		return
	}

	if len(resp.RequiredItems) > 0 {
		logger.Error(errors.Wrap(err, "failed to set all required items"))
		JSON(w, http.StatusBadRequest, resp)
		return
	}

	setAppConfigValuesResponse.Success = true
	JSON(w, http.StatusOK, setAppConfigValuesResponse)
}

func mergeConfigValues(config *kotsv1beta1.Config, existingValues *kotsv1beta1.ConfigValues, newValues *kotsv1beta1.ConfigValues) (*kotsv1beta1.ConfigValues, error) {
	unknownKeys := map[string]struct{}{}
	for k := range newValues.Spec.Values {
		unknownKeys[k] = struct{}{}
	}

	mergedValues := map[string]kotsv1beta1.ConfigValue{}
	for _, group := range config.Spec.Groups {
		for _, item := range group.Items {
			// process repeatable items
			for _, repeatGroup := range item.ValuesByGroup {
				for valueName := range repeatGroup {
					newValue, newOK := newValues.Spec.Values[valueName]
					existingValue, existingOK := existingValues.Spec.Values[valueName]
					if !newOK && !existingOK {
						continue
					}

					if existingOK {
						delete(unknownKeys, valueName)
					}

					if !newOK {
						mergedValues[valueName] = existingValue
						continue
					}

					mergedValues[valueName] = newValue
				}
			}

			newValue, newOK := newValues.Spec.Values[item.Name]
			existingValue, existingOK := existingValues.Spec.Values[item.Name]
			if !newOK && !existingOK {
				continue
			}

			if existingOK {
				delete(unknownKeys, item.Name)
			}

			if !newOK {
				mergedValues[item.Name] = existingValue
				continue
			}

			if item.Type == "password" && newValue.ValuePlaintext == "" {
				newValue.ValuePlaintext = newValue.Value
				newValue.Value = ""
			}

			mergedValues[item.Name] = newValue
		}
	}

	if len(unknownKeys) > 0 {
		return nil, errors.Errorf("new values contain unknown keys: %v", unknownKeys)
	}

	merged := &kotsv1beta1.ConfigValues{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "ConfigValues",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: existingValues.ObjectMeta.Name,
		},
		Spec: kotsv1beta1.ConfigValuesSpec{
			Values: mergedValues,
		},
	}

	return merged, nil
}

func updateConfigObject(config *kotsv1beta1.Config, configValues *kotsv1beta1.ConfigValues, merge bool) (*kotsv1beta1.Config, error) {
	newConfig := config.DeepCopy()

	for i, group := range newConfig.Spec.Groups {
		newItems := make([]kotsv1beta1.ConfigItem, 0)
		for _, item := range group.Items {

			replacementRepeatValues := map[string]string{}
			for valueName, value := range configValues.Spec.Values {
				if value.RepeatableItem == item.Name {
					replacementRepeatValues[valueName] = value.Value
				}
			}

			// ensure the map is initialized before we write to it
			if item.ValuesByGroup == nil {
				item.ValuesByGroup = map[string]kotsv1beta1.GroupValues{}
			}
			if len(replacementRepeatValues) > 0 {
				item.ValuesByGroup[group.Name] = replacementRepeatValues
			} else {
				item.ValuesByGroup = map[string]kotsv1beta1.GroupValues{}
			}

			newValue, ok := configValues.Spec.Values[item.Name]
			if !ok {
				if !merge {
					// this clears out values
					item.Value = multitype.BoolOrString{Type: item.Value.Type}
					item.Default = multitype.BoolOrString{Type: item.Value.Type}
				}
				newItems = append(newItems, item)
				continue
			}

			if newValue.Value != "" {
				newVal, err := item.Value.NewWithSameType(newValue.Value)
				if err != nil {
					return nil, errors.Wrap(err, "failed to update from Value")
				}
				item.Value = newVal
				item.Default = multitype.BoolOrString{Type: item.Value.Type}
			} else if newValue.ValuePlaintext != "" {
				newVal, err := item.Value.NewWithSameType(newValue.ValuePlaintext)
				if err != nil {
					return nil, errors.Wrap(err, "failed to update from ValuePlaintext")
				}
				item.Value = newVal
				item.Default = multitype.BoolOrString{Type: item.Value.Type}
			} else if newValue.Default != "" {
				newVal, err := item.Value.NewWithSameType(newValue.Default)
				if err != nil {
					return nil, errors.Wrap(err, "failed to update from Default")
				}
				item.Value = multitype.BoolOrString{Type: item.Value.Type}
				item.Default = newVal
			}
			newItems = append(newItems, item)
		}

		newConfig.Spec.Groups[i].Items = newItems
	}

	return newConfig, nil
}
