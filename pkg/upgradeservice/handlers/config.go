package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/config"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/kotsadmconfig"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
	configvalidation "github.com/replicatedhq/kots/pkg/kotsadmconfig/validation"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/render"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	"github.com/replicatedhq/kots/pkg/template"
	upgradepreflight "github.com/replicatedhq/kots/pkg/upgradeservice/preflight"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/multitype"
)

type CurrentAppConfigResponse struct {
	Success          bool                                     `json:"success"`
	Error            string                                   `json:"error,omitempty"`
	ConfigGroups     []kotsv1beta1.ConfigGroup                `json:"configGroups"`
	ValidationErrors []configtypes.ConfigGroupValidationError `json:"validationErrors,omitempty"`
}

type LiveAppConfigRequest struct {
	ConfigGroups []kotsv1beta1.ConfigGroup `json:"configGroups"`
}

type LiveAppConfigResponse struct {
	Success          bool                                     `json:"success"`
	Error            string                                   `json:"error,omitempty"`
	ConfigGroups     []kotsv1beta1.ConfigGroup                `json:"configGroups"`
	ValidationErrors []configtypes.ConfigGroupValidationError `json:"validationErrors,omitempty"`
}

type SaveAppConfigRequest struct {
	ConfigGroups []kotsv1beta1.ConfigGroup `json:"configGroups"`
}

type SaveAppConfigResponse struct {
	Success          bool                                     `json:"success"`
	Error            string                                   `json:"error,omitempty"`
	RequiredItems    []string                                 `json:"requiredItems,omitempty"`
	ValidationErrors []configtypes.ConfigGroupValidationError `json:"validationErrors,omitempty"`
}

type DownloadFileFromConfigResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) CurrentAppConfig(w http.ResponseWriter, r *http.Request) {
	currentAppConfigResponse := CurrentAppConfigResponse{
		Success: false,
	}

	params := GetContextParams(r)

	appLicense, err := kotsutil.LoadLicenseFromBytes([]byte(params.AppLicense))
	if err != nil {
		currentAppConfigResponse.Error = "failed to load license from bytes"
		logger.Error(errors.Wrap(err, currentAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(params.AppArchive)
	if err != nil {
		currentAppConfigResponse.Error = "failed to load kots kinds from path"
		logger.Error(errors.Wrap(err, currentAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	// get the non-rendered config from the upstream directory because we have to re-render it with the new values
	nonRenderedConfig, err := kotsutil.FindConfigInPath(filepath.Join(params.AppArchive, "upstream"))
	if err != nil {
		currentAppConfigResponse.Error = "failed to find non-rendered config"
		logger.Error(errors.Wrap(err, currentAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	registrySettings := registrytypes.RegistrySettings{
		Hostname:   params.RegistryEndpoint,
		Username:   params.RegistryUsername,
		Password:   params.RegistryPassword,
		Namespace:  params.RegistryNamespace,
		IsReadOnly: params.RegistryIsReadOnly,
	}

	// get values from saved app version
	configValues := map[string]template.ItemValue{}

	if kotsKinds.ConfigValues != nil {
		for key, value := range kotsKinds.ConfigValues.Spec.Values {
			generatedValue := template.ItemValue{
				Default:        value.Default,
				Value:          value.Value,
				Filename:       value.Filename,
				RepeatableItem: value.RepeatableItem,
			}
			configValues[key] = generatedValue
		}
	}

	versionInfo := template.VersionInfoFromInstallationSpec(params.NextSequence, params.AppIsAirgap, kotsKinds.Installation.Spec)
	appInfo := template.ApplicationInfo{Slug: params.AppSlug}
	renderedConfig, err := kotsconfig.TemplateConfigObjects(nonRenderedConfig, configValues, appLicense, &kotsKinds.KotsApplication, registrySettings, &versionInfo, &appInfo, kotsKinds.IdentityConfig, util.PodNamespace, false)
	if err != nil {
		logger.Error(err)
		currentAppConfigResponse.Error = "failed to render templates"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	currentAppConfigResponse.ConfigGroups = []kotsv1beta1.ConfigGroup{}
	if renderedConfig != nil {
		currentAppConfigResponse.ConfigGroups = renderedConfig.Spec.Groups
	}

	currentAppConfigResponse.Success = true
	JSON(w, http.StatusOK, currentAppConfigResponse)
}

func (h *Handler) LiveAppConfig(w http.ResponseWriter, r *http.Request) {
	liveAppConfigResponse := LiveAppConfigResponse{
		Success: false,
	}

	params := GetContextParams(r)

	appLicense, err := kotsutil.LoadLicenseFromBytes([]byte(params.AppLicense))
	if err != nil {
		liveAppConfigResponse.Error = "failed to load license from bytes"
		logger.Error(errors.Wrap(err, liveAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	liveAppConfigRequest := LiveAppConfigRequest{}
	if err := json.NewDecoder(r.Body).Decode(&liveAppConfigRequest); err != nil {
		logger.Error(err)
		liveAppConfigResponse.Error = "failed to decode request body"
		JSON(w, http.StatusBadRequest, liveAppConfigResponse)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(params.AppArchive)
	if err != nil {
		liveAppConfigResponse.Error = "failed to load kots kinds from path"
		logger.Error(errors.Wrap(err, liveAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	// get the non-rendered config from the upstream directory because we have to re-render it with the new values
	nonRenderedConfig, err := kotsutil.FindConfigInPath(filepath.Join(params.AppArchive, "upstream"))
	if err != nil {
		liveAppConfigResponse.Error = "failed to find non-rendered config"
		logger.Error(errors.Wrap(err, liveAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	registrySettings := registrytypes.RegistrySettings{
		Hostname:   params.RegistryEndpoint,
		Username:   params.RegistryUsername,
		Password:   params.RegistryPassword,
		Namespace:  params.RegistryNamespace,
		IsReadOnly: params.RegistryIsReadOnly,
	}

	// sequence +1 because the sequence will be incremented on save (and we want the preview to be accurate)
	configValues := configValuesFromConfigGroups(liveAppConfigRequest.ConfigGroups)
	versionInfo := template.VersionInfoFromInstallationSpec(params.NextSequence, params.AppIsAirgap, kotsKinds.Installation.Spec)
	appInfo := template.ApplicationInfo{Slug: params.AppSlug}

	renderedConfig, err := kotsconfig.TemplateConfigObjects(nonRenderedConfig, configValues, appLicense, &kotsKinds.KotsApplication, registrySettings, &versionInfo, &appInfo, kotsKinds.IdentityConfig, util.PodNamespace, false)
	if err != nil {
		liveAppConfigResponse.Error = "failed to render templates"
		logger.Error(errors.Wrap(err, liveAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	liveAppConfigResponse.ConfigGroups = []kotsv1beta1.ConfigGroup{}
	if renderedConfig != nil {
		validationErrors, err := configvalidation.ValidateConfigSpec(renderedConfig.Spec)
		if err != nil {
			liveAppConfigResponse.Error = "failed to validate config spec"
			logger.Error(errors.Wrap(err, liveAppConfigResponse.Error))
			JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
			return
		}

		liveAppConfigResponse.ConfigGroups = renderedConfig.Spec.Groups
		if len(validationErrors) > 0 {
			liveAppConfigResponse.ValidationErrors = validationErrors
			logger.Warnf("Validation errors found for config spec: %v", validationErrors)
		}
	}

	liveAppConfigResponse.Success = true
	JSON(w, http.StatusOK, liveAppConfigResponse)
}

func configValuesFromConfigGroups(configGroups []kotsv1beta1.ConfigGroup) map[string]template.ItemValue {
	configValues := map[string]template.ItemValue{}

	for _, group := range configGroups {
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

	return configValues
}

func (h *Handler) SaveAppConfig(w http.ResponseWriter, r *http.Request) {
	saveAppConfigResponse := SaveAppConfigResponse{
		Success: false,
	}

	params := GetContextParams(r)

	saveAppConfigRequest := SaveAppConfigRequest{}
	if err := json.NewDecoder(r.Body).Decode(&saveAppConfigRequest); err != nil {
		logger.Error(err)
		saveAppConfigResponse.Error = "failed to decode request body"
		JSON(w, http.StatusBadRequest, saveAppConfigResponse)
		return
	}

	validationErrors, err := configvalidation.ValidateConfigSpec(kotsv1beta1.ConfigSpec{Groups: saveAppConfigRequest.ConfigGroups})
	if err != nil {
		saveAppConfigResponse.Error = "failed to validate config spec."
		logger.Error(errors.Wrap(err, saveAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, saveAppConfigResponse)
		return
	}

	if len(validationErrors) > 0 {
		saveAppConfigResponse.Error = "invalid config values"
		saveAppConfigResponse.ValidationErrors = validationErrors
		logger.Errorf("%v, validation errors: %+v", saveAppConfigResponse.Error, validationErrors)
		JSON(w, http.StatusBadRequest, saveAppConfigResponse)
		return
	}

	requiredItems, requiredItemsTitles := kotsadmconfig.GetMissingRequiredConfig(saveAppConfigRequest.ConfigGroups)
	if len(requiredItems) > 0 {
		saveAppConfigResponse.RequiredItems = requiredItems
		saveAppConfigResponse.Error = fmt.Sprintf("The following fields are required: %s", strings.Join(requiredItemsTitles, ", "))
		logger.Errorf("%v, required items: %+v", saveAppConfigResponse.Error, requiredItems)
		JSON(w, http.StatusBadRequest, saveAppConfigResponse)
		return
	}

	registrySettings := registrytypes.RegistrySettings{
		Hostname:   params.RegistryEndpoint,
		Username:   params.RegistryUsername,
		Password:   params.RegistryPassword,
		Namespace:  params.RegistryNamespace,
		IsReadOnly: params.RegistryIsReadOnly,
	}

	app := &apptypes.App{
		ID:       params.AppID,
		Slug:     params.AppSlug,
		IsAirgap: params.AppIsAirgap,
		IsGitOps: params.AppIsGitOps,
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(params.AppArchive)
	if err != nil {
		saveAppConfigResponse.Error = "failed to load kots kinds from path"
		logger.Error(errors.Wrap(err, saveAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, saveAppConfigResponse)
		return
	}

	if kotsKinds.ConfigValues == nil {
		err = errors.New("config values not found")
		saveAppConfigResponse.Error = err.Error()
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, saveAppConfigResponse)
		return
	}

	values := kotsKinds.ConfigValues.Spec.Values
	kotsKinds.ConfigValues.Spec.Values = kotsadmconfig.UpdateAppConfigValues(values, saveAppConfigRequest.ConfigGroups)

	configValuesSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
	if err != nil {
		saveAppConfigResponse.Error = "failed to marshal config values"
		logger.Error(errors.Wrap(err, saveAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, saveAppConfigResponse)
		return
	}

	if err := os.WriteFile(filepath.Join(params.AppArchive, "upstream", "userdata", "config.yaml"), []byte(configValuesSpec), 0644); err != nil {
		saveAppConfigResponse.Error = "failed to write config values"
		logger.Error(errors.Wrap(err, saveAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, saveAppConfigResponse)
		return
	}

	err = render.RenderDir(rendertypes.RenderDirOptions{
		ArchiveDir:       params.AppArchive,
		App:              app,
		Downstreams:      []downstreamtypes.Downstream{{Name: "this-cluster"}},
		RegistrySettings: registrySettings,
		Sequence:         params.NextSequence,
		ReportingInfo:    params.ReportingInfo,
	})
	if err != nil {
		cause := errors.Cause(err)
		if _, ok := cause.(util.ActionableError); ok {
			saveAppConfigResponse.Error = err.Error()
			JSON(w, http.StatusInternalServerError, saveAppConfigResponse)
			return
		} else {
			saveAppConfigResponse.Error = "failed to render templates"
			logger.Error(errors.Wrap(err, saveAppConfigResponse.Error))
			JSON(w, http.StatusInternalServerError, saveAppConfigResponse)
			return
		}
	}

	if err := upgradepreflight.Run(params); err != nil {
		saveAppConfigResponse.Error = "failed to run preflights"
		logger.Error(errors.Wrap(err, saveAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, saveAppConfigResponse)
		return
	}

	saveAppConfigResponse.Success = true
	JSON(w, http.StatusOK, saveAppConfigResponse)
}

func (h *Handler) DownloadFileFromConfig(w http.ResponseWriter, r *http.Request) {
	downloadFileFromConfigResponse := DownloadFileFromConfigResponse{
		Success: false,
	}

	params := GetContextParams(r)

	filename := mux.Vars(r)["filename"]
	if filename == "" {
		logger.Error(errors.New("filename parameter is empty"))
		downloadFileFromConfigResponse.Error = "failed to parse filename, parameter was empty"
		JSON(w, http.StatusInternalServerError, downloadFileFromConfigResponse)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(params.AppArchive)
	if err != nil {
		downloadFileFromConfigResponse.Error = "failed to load kots kinds from path"
		logger.Error(errors.Wrap(err, downloadFileFromConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, downloadFileFromConfigResponse)
		return
	}

	var configValue *string
	for _, v := range kotsKinds.ConfigValues.Spec.Values {
		if v.Filename == filename {
			configValue = &v.Value
			break
		}
	}
	if configValue == nil {
		logger.Error(errors.New("could not find requested file"))
		downloadFileFromConfigResponse.Error = "could not find requested file"
		JSON(w, http.StatusInternalServerError, downloadFileFromConfigResponse)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(*configValue)
	if err != nil {
		logger.Error(err)
		downloadFileFromConfigResponse.Error = "failed to decode config value"
		JSON(w, http.StatusInternalServerError, downloadFileFromConfigResponse)
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(decoded)))
	w.WriteHeader(http.StatusOK)
	w.Write(decoded)
}
