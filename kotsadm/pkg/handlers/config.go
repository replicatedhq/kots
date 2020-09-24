package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	kotsadmconfig "github.com/replicatedhq/kots/kotsadm/pkg/config"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/render"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	versiontypes "github.com/replicatedhq/kots/kotsadm/pkg/version/types"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/template"
)

type UpdateAppConfigRequest struct {
	Sequence         int64                      `json:"sequence"`
	CreateNewVersion bool                       `json:"createNewVersion"`
	ConfigGroups     []*kotsv1beta1.ConfigGroup `json:"configGroups"`
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

func UpdateAppConfig(w http.ResponseWriter, r *http.Request) {
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

	if !updateAppConfigRequest.CreateNewVersion {
		// special case handling for "do not create new version"
		// no need to update versions after this/search for latest sequence that has the same upstream version/etc
		resp, err := updateAppConfig(foundApp, updateAppConfigRequest.Sequence, updateAppConfigRequest, true)
		if err != nil {
			logger.Error(err)
			JSON(w, http.StatusInternalServerError, resp)
			return
		}

		if len(resp.RequiredItems) > 0 {
			JSON(w, http.StatusBadRequest, resp)
			return
		}

		JSON(w, http.StatusOK, resp)
		return
	}

	// find the update cursor referred to by updateAppConfigRequest.Sequence
	// then find the latest sequence with that update cursor, and use that for the update we're making here
	// (for instance, the registry settings may have changed)
	// if there are additional update cursors past this, also create updates for them
	latestSequenceMatchingUpdateCursor, laterVersions, err := getLaterVersions(foundApp, updateAppConfigRequest.Sequence)
	if err != nil {
		logger.Error(err)
		updateAppConfigResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, updateAppConfigResponse)
		return
	}

	// attempt to apply the config to the app version specified in the request
	resp, err := updateAppConfig(foundApp, latestSequenceMatchingUpdateCursor, updateAppConfigRequest, true)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, resp)
		return
	}

	if len(resp.RequiredItems) > 0 {
		JSON(w, http.StatusBadRequest, resp)
		return
	}

	// if there were no errors applying the config for the desired version, do the same for any later versions too
	for _, version := range laterVersions {
		_, err := updateAppConfig(foundApp, version.Sequence, updateAppConfigRequest, false)
		if err != nil {
			logger.Error(errors.Wrapf(err, "error creating app with new config based on sequence %d for upstream %q", version.Sequence, version.KOTSKinds.Installation.Spec.VersionLabel))
		}
	}

	JSON(w, http.StatusOK, UpdateAppConfigResponse{Success: true})
}

func LiveAppConfig(w http.ResponseWriter, r *http.Request) {
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

	archiveDir, err := store.GetStore().GetAppVersionArchive(foundApp.ID, liveAppConfigRequest.Sequence)
	if err != nil {
		liveAppConfigResponse.Error = "failed to get app version archive"
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}
	defer os.RemoveAll(archiveDir)

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		liveAppConfigResponse.Error = "failed to load kots kinds from path"
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	// get values from request
	configValues := map[string]template.ItemValue{}

	for _, group := range liveAppConfigRequest.ConfigGroups {
		for _, item := range group.Items {
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
			configValues[item.Name] = generatedValue
		}
	}

	registryInfo, err := store.GetStore().GetRegistryDetailsForApp(foundApp.ID)
	if err != nil {
		liveAppConfigResponse.Error = "failed to get app registry info"
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	localRegistry := template.LocalRegistry{}
	if registryInfo != nil {
		localRegistry.Host = registryInfo.Hostname
		localRegistry.Namespace = registryInfo.Namespace
		localRegistry.Username = registryInfo.Username
		localRegistry.Password = registryInfo.Password
	}

	versionInfo := template.VersionInfoFromInstallation(liveAppConfigRequest.Sequence+1, foundApp.IsAirgap, kotsKinds.Installation.Spec) // sequence +1 because the sequence will be incremented on save (and we want the preview to be accurate)
	renderedConfig, err := kotsconfig.TemplateConfigObjects(kotsKinds.Config, configValues, appLicense, localRegistry, &versionInfo)
	if err != nil {
		liveAppConfigResponse.Error = "failed to render templates"
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	JSON(w, http.StatusOK, LiveAppConfigResponse{Success: true, ConfigGroups: renderedConfig.Spec.Groups})
}

func CurrentAppConfig(w http.ResponseWriter, r *http.Request) {
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

	archiveDir, err := store.GetStore().GetAppVersionArchive(foundApp.ID, int64(sequence))
	if err != nil {
		currentAppConfigResponse.Error = "failed to get app version archive"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}
	defer os.RemoveAll(archiveDir)

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		currentAppConfigResponse.Error = "failed to load kots kinds from path"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	// get values from saved app version
	configValues := map[string]template.ItemValue{}

	for key, value := range kotsKinds.ConfigValues.Spec.Values {
		generatedValue := template.ItemValue{
			Default: value.Default,
			Value:   value.Value,
		}
		configValues[key] = generatedValue
	}

	registryInfo, err := store.GetStore().GetRegistryDetailsForApp(foundApp.ID)
	if err != nil {
		currentAppConfigResponse.Error = "failed to get app registry info"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	localRegistry := template.LocalRegistry{}
	if registryInfo != nil {
		localRegistry.Host = registryInfo.Hostname
		localRegistry.Namespace = registryInfo.Namespace
		localRegistry.Username = registryInfo.Username
		localRegistry.Password = registryInfo.Password
	}

	versionInfo := template.VersionInfoFromInstallation(int64(sequence)+1, foundApp.IsAirgap, kotsKinds.Installation.Spec) // sequence +1 because the sequence will be incremented on save (and we want the preview to be accurate)
	renderedConfig, err := kotsconfig.TemplateConfigObjects(kotsKinds.Config, configValues, appLicense, localRegistry, &versionInfo)
	if err != nil {
		currentAppConfigResponse.Error = "failed to render templates"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	JSON(w, http.StatusOK, CurrentAppConfigResponse{Success: true, ConfigGroups: renderedConfig.Spec.Groups})
}

// if isPrimaryVersion is false, missing a required config field will not cause a failure, and instead will create
// the app version with status needs_config
func updateAppConfig(updateApp *apptypes.App, sequence int64, req UpdateAppConfigRequest, isPrimaryVersion bool) (UpdateAppConfigResponse, error) {
	updateAppConfigResponse := UpdateAppConfigResponse{
		Success: false,
	}

	archiveDir, err := store.GetStore().GetAppVersionArchive(updateApp.ID, sequence)
	if err != nil {
		updateAppConfigResponse.Error = "failed to get app version archive"
		return updateAppConfigResponse, err
	}
	defer os.RemoveAll(archiveDir)

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		updateAppConfigResponse.Error = "failed to load kots kinds from path"
		return updateAppConfigResponse, err
	}

	// check for unset required items
	requiredItems := make([]string, 0, 0)
	requiredItemsTitles := make([]string, 0, 0)
	for _, group := range req.ConfigGroups {
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
	for _, group := range req.ConfigGroups {
		for _, item := range group.Items {
			if item.Value.Type == multitype.Bool {
				updatedValue := item.Value.BoolVal
				v := values[item.Name]
				v.Value = strconv.FormatBool(updatedValue)
				values[item.Name] = v
			} else if item.Value.Type == multitype.String {
				updatedValue := item.Value.String()
				if item.Type == "password" {
					// encrypt using the key
					cipher, err := crypto.AESCipherFromString(kotsKinds.Installation.Spec.EncryptionKey)
					if err != nil {
						updateAppConfigResponse.Error = "failed to get encryption cipher"
						return updateAppConfigResponse, err
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
		}
	}

	if kotsKinds.ConfigValues == nil {
		updateAppConfigResponse.Error = "no config values found"
		return updateAppConfigResponse, errors.New("no config values found")
	}

	kotsKinds.ConfigValues.Spec.Values = values

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

	err = render.RenderDir(archiveDir, app, downstreams, registrySettings)
	if err != nil {
		updateAppConfigResponse.Error = "failed to render archive directory"
		return updateAppConfigResponse, err
	}

	if req.CreateNewVersion {
		newSequence, err := version.CreateVersion(updateApp.ID, archiveDir, "Config Change", updateApp.CurrentSequence)
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

	if err := downstream.SetDownstreamVersionPendingPreflight(updateApp.ID, int64(sequence)); err != nil {
		updateAppConfigResponse.Error = "failed to set downstream status to 'pending preflight'"
		return updateAppConfigResponse, err
	}

	if err := preflight.Run(updateApp.ID, int64(sequence), updateApp.IsAirgap, archiveDir); err != nil {
		updateAppConfigResponse.Error = errors.Cause(err).Error()
		return updateAppConfigResponse, err
	}

	updateAppConfigResponse.Success = true
	return updateAppConfigResponse, nil
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

func getLaterVersions(versionedApp *apptypes.App, startSequence int64) (int64, []versiontypes.AppVersion, error) {
	thisAppVersion, err := store.GetStore().GetAppVersion(versionedApp.ID, startSequence)
	if err != nil {
		return -1, nil, errors.Wrap(err, "failed to get this appversion")
	}

	laterAppVersions, err := store.GetStore().GetAppVersionsAfter(versionedApp.ID, startSequence)
	if err != nil {
		return -1, nil, errors.Wrap(err, "failed to get later app versions")
	}

	// latestSequenceWithThisUpdateCursor is the newest local version
	// of the same upstream version
	latestSequenceWithThisUpdateCursor := thisAppVersion.Sequence
	for _, laterAppVersion := range laterAppVersions {
		if laterAppVersion.KOTSKinds.Installation.Spec.UpdateCursor == thisAppVersion.KOTSKinds.Installation.Spec.UpdateCursor {
			if laterAppVersion.Sequence > latestSequenceWithThisUpdateCursor {
				latestSequenceWithThisUpdateCursor = laterAppVersion.Sequence
			}
		}
	}

	laterVersions := map[string][]versiontypes.AppVersion{}
	for _, laterAppVersion := range laterAppVersions {
		if current, ok := laterVersions[laterAppVersion.KOTSKinds.Installation.Spec.UpdateCursor]; ok {
			current = append(current, *laterAppVersion)
			laterVersions[laterAppVersion.KOTSKinds.Installation.Spec.UpdateCursor] = current
		} else {
			laterVersions[laterAppVersion.KOTSKinds.Installation.Spec.UpdateCursor] = []versiontypes.AppVersion{
				*laterAppVersion,
			}
		}
	}

	// ensure that the returned versions array is sorted
	keys := []string{}
	for key := range laterVersions {
		keys = append(keys, key)
	}

	// TODO sort by something in kotsutil that i need to write (these are either ints or semvers)
	sort.Strings(keys)

	sortedVersions := []versiontypes.AppVersion{}
	for _, key := range keys {
		sortedVersions = append(sortedVersions, laterVersions[key]...)
	}

	return latestSequenceWithThisUpdateCursor, sortedVersions, nil
}
