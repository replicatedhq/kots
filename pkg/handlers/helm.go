package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	kotsbase "github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/helm"
	kotshelm "github.com/replicatedhq/kots/pkg/helm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	yaml "github.com/replicatedhq/yaml/v3"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
)

// IsHelmManagedResponse - response body for the is helm managed endpoint
type IsHelmManagedResponse struct {
	Success       bool `json:"success"`
	IsHelmManaged bool `json:"isHelmManaged"`
}

type GetAppValuesFileResponse struct {
	Success bool `json:"success"`
}

//  IsHelmManaged - report whether or not kots is running in helm managed mode
func (h *Handler) IsHelmManaged(w http.ResponseWriter, r *http.Request) {
	helmManagedResponse := IsHelmManagedResponse{
		Success:       true,
		IsHelmManaged: util.IsHelmManaged(),
	}

	JSON(w, http.StatusOK, helmManagedResponse)
}

func (h *Handler) GetAppValuesFile(w http.ResponseWriter, r *http.Request) {
	getAppValuesFileResponse := GetAppValuesFileResponse{
		Success: false,
	}
	appSlug := mux.Vars(r)["appSlug"]
	helmApp := helm.GetHelmApp(appSlug)
	if helmApp == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	appSecret, err := helm.GetChartConfigSecret(helmApp)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get secret"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// if there is no config secret then app is not configurable
	if appSecret == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	configValues, err := helm.GetTempConfigValues(helmApp)
	if err != nil && !kuberneteserrors.IsNotFound(errors.Cause(err)) {
		logger.Error(errors.Wrap(err, "failed to get temp config values"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	helmChart, err := kotsbase.ParseHelmChart(appSecret.Data["chart"])
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

	kotsKinds, err := helm.GetKotsKinds(helmApp)
	kotsKinds.ConfigValues = configValues

	renderedValues, err := kotshelm.RenderValuesFromConfig(helmApp, &kotsKinds, appSecret.Data["chart"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// get a intersected map containing tmplVals keys with renderedValues values
	intersectVals := kotsv1beta1.GetMapIntersect(tmplVals, renderedValues)

	mergedHelmValues, err := kotshelm.GetMergedValues(helmApp.Release.Chart.Values, intersectVals)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := yaml.Marshal(mergedHelmValues)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	getAppValuesFileResponse.Success = true
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-values.yaml", appSlug))
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(http.StatusOK)
	w.Write(b)
	JSON(w, http.StatusOK, getAppValuesFileResponse)
}

func getCompatibleAppFromHelmApp(helmApp *apptypes.HelmApp) (*apptypes.App, error) {
	chartApp, err := responseAppFromHelmApp(helmApp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert release to app")
	}

	foundApp := &apptypes.App{ID: chartApp.ID, Slug: chartApp.Slug, Name: chartApp.Name}
	return foundApp, nil
}
