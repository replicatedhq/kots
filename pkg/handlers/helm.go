package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/helm"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
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
		Success: false,
	}

	var err error
	isHelmManaged := false

	isHelmManagedStr := os.Getenv("IS_HELM_MANAGED")
	if isHelmManagedStr != "" {
		isHelmManaged, err = strconv.ParseBool(isHelmManagedStr)
		if err != nil {
			err = errors.Wrap(err, "failed to convert IS_HELM_MANAGED env var to bool")
			logger.Error(err)
			helmManagedResponse.Success = false
			JSON(w, http.StatusInternalServerError, helmManagedResponse)
			return
		}
	}

	helmManagedResponse.IsHelmManaged = isHelmManaged
	helmManagedResponse.Success = true
	JSON(w, http.StatusOK, helmManagedResponse)
}

func (h *Handler) GetAppValuesFile(w http.ResponseWriter, r *http.Request) {
	getAppValuesFileResponse := GetAppValuesFileResponse{
		Success: false,
	}
	appSlug := mux.Vars(r)["appSlug"]
	release := helm.GetHelmRelease(appSlug)
	if release == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	dat, err := os.ReadFile(release.PathToValuesFile)
	if err != nil {
		err = errors.Wrap(err, "failed to read values file")
		logger.Error(err)
		getAppValuesFileResponse.Success = false
		JSON(w, http.StatusInternalServerError, getAppValuesFileResponse)
		return
	}

	getAppValuesFileResponse.Success = true
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-values.yaml", appSlug))
	w.Header().Set("Content-Length", strconv.Itoa(len(dat)))
	w.WriteHeader(http.StatusOK)
	w.Write(dat)
	JSON(w, http.StatusOK, getAppValuesFileResponse)
}

func getLicenseForHelmApp(chartName string) (*kotsv1beta1.License, *apptypes.App, error) {
	release := helm.GetHelmRelease(chartName)
	if release == nil {
		return nil, nil, errors.Errorf("chart %q is not found in cache", chartName)
	}

	chartApp, err := responseAppFromHelmApp(release)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to convert release to app")
	}

	foundApp := &apptypes.App{ID: chartApp.ID, Slug: chartApp.Slug, Name: chartApp.Name}
	apiEndpoint := os.Getenv("REPLICATED_API_ENDPOINT")

	// get license
	req, err := util.NewRequest(http.MethodGet, fmt.Sprintf("%s/license", apiEndpoint), nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create new HTTP request")
	}
	var licId string
	if replicatedValues, _ := release.Release.Chart.Values["replicated"].(map[string]interface{}); replicatedValues != nil {
		licId = replicatedValues["license_id"].(string)
	}
	if licId == "" {
		return nil, nil, errors.New("replicated license id not present in values")
	}
	req.SetBasicAuth(licId, licId)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to perform HTTP request")
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, nil, errors.New(fmt.Sprintf("failed to perform http request, got non 200 status code of: %v", resp.StatusCode))
	}
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read response body")
	}

	license, err := kotsutil.LoadLicenseFromBytes(responseBody)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load license from response body bytes")
	}

	return license, foundApp, nil
}
