package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/helm"
	kotshelm "github.com/replicatedhq/kots/pkg/helm"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/util"
	yaml "github.com/replicatedhq/yaml/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsHelmManagedResponse - response body for the is helm managed endpoint
type IsHelmManagedResponse struct {
	Success       bool `json:"success"`
	IsHelmManaged bool `json:"isHelmManaged"`
}

// IsHelmManaged - report whether or not kots is running in helm managed mode
func (h *Handler) IsHelmManaged(w http.ResponseWriter, r *http.Request) {
	helmManagedResponse := IsHelmManagedResponse{
		Success:       true,
		IsHelmManaged: util.IsHelmManaged(),
	}

	JSON(w, http.StatusOK, helmManagedResponse)
}

func (h *Handler) GetAppValuesFile(w http.ResponseWriter, r *http.Request) {
	if !util.IsHelmManaged() {
		logger.Errorf("values file can only be dowloaded in Helm managed mode")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.ParseInt(mux.Vars(r)["sequence"], 10, 64)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse app sequence"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	helmApp := helm.GetHelmApp(appSlug)
	if helmApp == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var kotsKinds *kotsutil.KotsKinds
	var tmplVals map[string]interface{}
	var helmChartFile []byte

	isPending, _ := strconv.ParseBool(r.URL.Query().Get("isPending"))
	if isPending {
		licenseID := helm.GetKotsLicenseID(&helmApp.Release)
		if licenseID == "" {
			logger.Error(errors.Errorf("no license and no license ID found for release %s", helmApp.Release.Name))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		secret, err := helm.GetReplicatedSecretFromUpstreamChartVersion(helmApp, licenseID, r.URL.Query().Get("semver"))
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get replicated secret from upstream chart"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		helmChartFile = secret.Data["chart"]

		k, err := helm.GetKotsKindsFromUpstreamChartVersion(helmApp, licenseID, r.URL.Query().Get("semver"))
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get kotskinds from upstream chart"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		licenseData, err := replicatedapp.GetLatestLicenseForHelm(licenseID)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to download license for chart archive"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		k.License = licenseData.License

		kotsKinds = &k
	} else {
		replicatedSecret, err := helm.GetReplicatedSecretForRevision(helmApp.Release.Name, sequence, helmApp.Namespace)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get secret"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		helmChartFile = replicatedSecret.Data["chart"]

		k, err := helm.GetKotsKindsForRevision(helmApp.Release.Name, sequence, helmApp.Namespace)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get kots kinds for helm"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		kotsKinds = &k
	}

	helmChart, err := kotsutil.LoadV1Beta1HelmChartFromContents(helmChartFile)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse HelmChart file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tmplVals, err = helmChart.Spec.GetReplTmplValues(helmChart.Spec.Values)
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
			Values: helmApp.TempConfigValues,
		},
	}

	renderedValues, err := kotshelm.RenderValuesFromConfig(helmApp, kotsKinds, helmChartFile)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get render values from config"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// get a intersected map containing tmplVals keys with renderedValues values
	intersectVals := kotsv1beta1.GetMapIntersect(tmplVals, renderedValues)

	mergedHelmValues, err := kotshelm.GetMergedValues(helmApp.Release.Chart.Values, intersectVals)
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
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-values.yaml", appSlug))
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func getCompatibleAppFromHelmApp(helmApp *apptypes.HelmApp) (*apptypes.App, error) {
	chartApp, err := helm.ResponseAppFromHelmApp(helmApp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert release to app")
	}

	foundApp := &apptypes.App{ID: chartApp.ID, Slug: chartApp.Slug, Name: chartApp.Name}
	return foundApp, nil
}
