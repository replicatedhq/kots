package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/api/handlers/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	kotsbase "github.com/replicatedhq/kots/pkg/base"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/helm"
	kotshelm "github.com/replicatedhq/kots/pkg/helm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/util"
	yaml "github.com/replicatedhq/yaml/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ apptypes.AppType = (*FakeApp)(nil)

type FakeApp struct {
	ChartName string
}

func (a *FakeApp) GetID() string {
	return a.ChartName
}
func (a *FakeApp) GetSlug() string {
	return a.ChartName
}
func (a *FakeApp) GetCurrentSequence() int64 {
	return 0 // TODO
}
func (a *FakeApp) GetIsAirgap() bool {
	return false // TODO
}
func (a *FakeApp) GetNamespace() string {
	return "default" // TODO
}

func (h *HelmConfigHandler) Ping(w http.ResponseWriter, r *http.Request) {
	pingResponse := PingResponse{
		Ping: "pong",
	}
	JSON(w, 200, pingResponse)
}

//  IsHelmManaged - report whether or not kots is running in helm managed mode
func (h *HelmConfigHandler) IsHelmManaged(w http.ResponseWriter, r *http.Request) {
	helmManagedResponse := IsHelmManagedResponse{
		Success:       true,
		IsHelmManaged: util.IsHelmManaged(),
	}

	JSON(w, http.StatusOK, helmManagedResponse)
}

//  IsInitialConfigMode - report whether or not kots is running in helm managed mode
func (h *HelmConfigHandler) IsInitialConfigMode(w http.ResponseWriter, r *http.Request) {
	response := IsInitialConfigModeResponse{
		Success:             true,
		IsInitialConfigMode: util.IsInitialConfigMode(),
		VersionLabel:        h.ChartVersion,
	}

	JSON(w, http.StatusOK, response)
}

func (h *HelmConfigHandler) GetApp(w http.ResponseWriter, r *http.Request) {
	if !util.IsHelmManaged() || !util.IsInitialConfigMode() {
		logger.Errorf("this handler can only be used for initial helm config")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	responseApp := types.HelmResponseApp{
		ResponseApp: types.ResponseApp{
			ID:             h.ChartName,
			Slug:           h.ChartName,
			Name:           h.ChartName,
			Namespace:      "default",
			IsConfigurable: true,
		},
		ChartPath: h.ChartPath,
	}

	JSON(w, http.StatusOK, responseApp)
}

func (h *HelmConfigHandler) LiveAppConfig(w http.ResponseWriter, r *http.Request) {
	liveAppConfigResponse := LiveAppConfigResponse{
		Success: false,
	}

	if !util.IsHelmManaged() || !util.IsInitialConfigMode() {
		logger.Errorf("this handler can only be used for initial helm config")
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	liveAppConfigRequest := LiveAppConfigRequest{}
	if err := json.NewDecoder(r.Body).Decode(&liveAppConfigRequest); err != nil {
		liveAppConfigResponse.Error = "failed to decode request body"
		logger.Error(errors.Wrap(err, liveAppConfigResponse.Error))
		JSON(w, http.StatusBadRequest, liveAppConfigResponse)
		return
	}

	var localRegistry template.LocalRegistry
	app := &FakeApp{
		ChartName: h.ChartName,
	}
	configValues := configValuesFromConfigGroups(liveAppConfigRequest.ConfigGroups)

	filename := filepath.Join(h.ChartRootDir, h.ChartName, "templates", "_replicated", "secret.yaml")
	secretData, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get read secret data from chart template"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	kotsKinds, err := helm.GetKotsKindsFromUpstreamSecretData(secretData)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get kotskinds from upstream secret data"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	versionInfo := template.VersionInfoFromInstallation(liveAppConfigRequest.Sequence+1, app.GetIsAirgap(), kotsKinds.Installation.Spec) // sequence +1 because the sequence will be incremented on save (and we want the preview to be accurate)
	appInfo := template.ApplicationInfo{Slug: app.GetSlug()}
	renderedConfig, err := kotsconfig.TemplateConfigObjects(kotsKinds.Config, configValues, kotsKinds.License, &kotsKinds.KotsApplication, localRegistry, &versionInfo, &appInfo, kotsKinds.IdentityConfig, app.GetNamespace(), false)
	if err != nil {
		logger.Error(err)
		liveAppConfigResponse.Error = "failed to render templates"
		JSON(w, http.StatusInternalServerError, liveAppConfigResponse)
		return
	}

	liveAppConfigResponse.Success = true
	if renderedConfig == nil {
		liveAppConfigResponse.ConfigGroups = []kotsv1beta1.ConfigGroup{}
	} else {
		liveAppConfigResponse.ConfigGroups = renderedConfig.Spec.Groups
	}

	JSON(w, http.StatusOK, liveAppConfigResponse)
}

func (h *HelmConfigHandler) UpdateAppConfig(w http.ResponseWriter, r *http.Request) {
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

	requiredItems, requiredItemsTitles := getMissingRequiredConfig(updateAppConfigRequest.ConfigGroups)
	if len(requiredItems) > 0 {
		updateAppConfigResponse.RequiredItems = requiredItems
		updateAppConfigResponse.Error = fmt.Sprintf("The following fields are required: %s", strings.Join(requiredItemsTitles, ", "))
		JSON(w, http.StatusBadRequest, updateAppConfigResponse)
		return
	}

	h.TempConfigValues = updateAppConfigValues(h.TempConfigValues, updateAppConfigRequest.ConfigGroups)
	h.ConfigValuesSaved = true

	updateAppConfigResponse.Success = true
	JSON(w, http.StatusOK, updateAppConfigResponse)
}

func (h *HelmConfigHandler) GetAppValuesFile(w http.ResponseWriter, r *http.Request) {
	if !util.IsHelmManaged() || !util.IsInitialConfigMode() {
		logger.Errorf("this handler can only be used for initial helm config")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !h.ConfigValuesSaved {
		err := errors.New("config file cannot be downloaded because config has not been saved")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	app := &FakeApp{
		ChartName: h.ChartName,
	}

	secretFilename := filepath.Join(h.ChartRootDir, h.ChartName, "templates", "_replicated", "secret.yaml")
	secretData, err := ioutil.ReadFile(secretFilename)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get read secret data from chart template"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	secret, err := helm.GetKotsSecretFromUpstreamSecretData(secretData)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get replicated secret from upstream chart"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	helmChartFile := secret.Data["chart"]

	kotsKinds, err := helm.GetKotsKindsFromReplicatedSecret(secret)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get kotskinds from upstream chart"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	helmChart, err := kotsbase.ParseHelmChart(helmChartFile)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse HelmChart file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tmplVals, err := helmChart.Spec.GetReplTmplValues(helmChart.Spec.Values)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get templated values"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// keeping this assignment out of GetKotsKindsFromHelmApp because this is specific to file download endpoint
	kotsKinds.ConfigValues = &kotsv1beta1.ConfigValues{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "ConfigValues",
		},
		Spec: kotsv1beta1.ConfigValuesSpec{
			Values: h.TempConfigValues,
		},
	}

	renderedValues, err := kotshelm.RenderValuesFromConfig(app, &kotsKinds, helmChartFile)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get render values from config"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// get a intersected map containing tmplVals keys with renderedValues values
	intersectVals := kotsv1beta1.GetMapIntersect(tmplVals, renderedValues)

	valuesFilename := filepath.Join(h.ChartRootDir, h.ChartName, "values.yaml")
	valuesData, err := ioutil.ReadFile(valuesFilename)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to read values.yaml"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defaultChartValues := map[string]interface{}{}
	err = yaml.Unmarshal(valuesData, defaultChartValues)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to unmarshal values data"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	mergedHelmValues, err := kotshelm.GetMergedValues(defaultChartValues, intersectVals)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to merge values with templated values"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if kotsKinds.ConfigValues != nil {
		v, err := kotshelm.GetConfigValuesMap(kotsKinds.ConfigValues)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get app config values sub-map"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		m, err := kotshelm.GetMergedValues(mergedHelmValues, v)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to merge app config to helm values"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		mergedHelmValues = m
	}

	b, err := yaml.Marshal(mergedHelmValues)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-values.yaml", app.GetSlug()))
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func (h *HelmConfigHandler) GetInitialAppConfig(w http.ResponseWriter, r *http.Request) {
	currentAppConfigResponse := CurrentAppConfigResponse{
		Success: false,
	}

	if !util.IsHelmManaged() || !util.IsInitialConfigMode() {
		logger.Errorf("this handler can only be used for initial helm config")
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	var localRegistry template.LocalRegistry
	app := FakeApp{
		ChartName: h.ChartName,
	}
	configGroups := []kotsv1beta1.ConfigGroup{}

	filename := filepath.Join(h.ChartRootDir, h.ChartName, "templates", "_replicated", "secret.yaml")
	secretData, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get read secret data from chart template"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	kotsKinds, err := helm.GetKotsKindsFromUpstreamSecretData(secretData)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get kotskinds from upstream secret data"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

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

	versionInfo := template.VersionInfoFromInstallation(1, app.GetIsAirgap(), kotsKinds.Installation.Spec)
	appInfo := template.ApplicationInfo{Slug: app.GetSlug()}
	renderedConfig, err := kotsconfig.TemplateConfigObjects(kotsKinds.Config, configValues, kotsKinds.License, &kotsKinds.KotsApplication, localRegistry, &versionInfo, &appInfo, kotsKinds.IdentityConfig, app.GetNamespace(), false)
	if err != nil {
		logger.Error(err)
		currentAppConfigResponse.Error = "failed to render templates"
		JSON(w, http.StatusInternalServerError, currentAppConfigResponse)
		return
	}

	if renderedConfig != nil {
		configGroups = renderedConfig.Spec.Groups
	}

	currentAppConfigResponse.Success = true
	currentAppConfigResponse.ConfigGroups = configGroups
	currentAppConfigResponse.DownstreamVersion = nil
	JSON(w, http.StatusOK, currentAppConfigResponse)
}
