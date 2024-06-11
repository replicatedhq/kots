package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/config"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
	configvalidation "github.com/replicatedhq/kots/pkg/kotsadmconfig/validation"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/template"
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

func (h *Handler) CurrentAppConfig(w http.ResponseWriter, r *http.Request) {
	currentAppConfigResponse := CurrentAppConfigResponse{
		Success: false,
	}

	params := GetContextParams(r)
	appSlug := mux.Vars(r)["appSlug"]

	if params.AppSlug != appSlug {
		currentAppConfigResponse.Error = "app slug does not match"
		JSON(w, http.StatusForbidden, currentAppConfigResponse)
		return
	}

	appLicense, err := kotsutil.LoadLicenseFromBytes([]byte(params.AppLicense))
	if err != nil {
		currentAppConfigResponse.Error = "failed to load license from bytes"
		logger.Error(errors.Wrap(err, currentAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(params.BaseArchive) // TODO NOW: rename BaseArchive
	if err != nil {
		currentAppConfigResponse.Error = "failed to load kots kinds from path"
		logger.Error(errors.Wrap(err, currentAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	// get the non-rendered config from the upstream directory because we have to re-render it with the new values
	nonRenderedConfig, err := kotsutil.FindConfigInPath(filepath.Join(params.BaseArchive, "upstream"))
	if err != nil {
		currentAppConfigResponse.Error = "failed to find non-rendered config"
		logger.Error(errors.Wrap(err, currentAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	localRegistry := registrytypes.RegistrySettings{
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
	renderedConfig, err := kotsconfig.TemplateConfigObjects(nonRenderedConfig, configValues, appLicense, &kotsKinds.KotsApplication, localRegistry, &versionInfo, &appInfo, kotsKinds.IdentityConfig, util.PodNamespace, false)
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
	appSlug := mux.Vars(r)["appSlug"]

	if params.AppSlug != appSlug {
		liveAppConfigResponse.Error = "app slug does not match"
		JSON(w, http.StatusForbidden, liveAppConfigResponse)
		return
	}

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

	kotsKinds, err := kotsutil.LoadKotsKinds(params.BaseArchive)
	if err != nil {
		liveAppConfigResponse.Error = "failed to load kots kinds from path"
		logger.Error(errors.Wrap(err, liveAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	// get the non-rendered config from the upstream directory because we have to re-render it with the new values
	nonRenderedConfig, err := kotsutil.FindConfigInPath(filepath.Join(params.BaseArchive, "upstream"))
	if err != nil {
		liveAppConfigResponse.Error = "failed to find non-rendered config"
		logger.Error(errors.Wrap(err, liveAppConfigResponse.Error))
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	localRegistry := registrytypes.RegistrySettings{
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

	renderedConfig, err := kotsconfig.TemplateConfigObjects(nonRenderedConfig, configValues, appLicense, &kotsKinds.KotsApplication, localRegistry, &versionInfo, &appInfo, kotsKinds.IdentityConfig, util.PodNamespace, false)
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
