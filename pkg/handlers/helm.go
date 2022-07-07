package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
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
	app := mux.Vars(r)["appSlug"]
	appCache := getHelmAppCache()
	helmApp := appCache[app]

	dat, err := os.ReadFile(helmApp.PathToValuesFile)
	if err != nil {
		err = errors.Wrap(err, "failed to read values file")
		logger.Error(err)
		getAppValuesFileResponse.Success = false
		JSON(w, http.StatusInternalServerError, getAppValuesFileResponse)
		return
	}

	getAppValuesFileResponse.Success = true
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-values.yaml", app))
	w.Header().Set("Content-Length", strconv.Itoa(len(dat)))
	w.WriteHeader(http.StatusOK)
	w.Write(dat)
	JSON(w, http.StatusOK, getAppValuesFileResponse)
}
