package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/helm"
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
	release := helm.GetHelmApp(appSlug)
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

func getCompatibleAppFromHelmApp(helmApp *helm.HelmApp) (*apptypes.App, error) {
	chartApp, err := responseAppFromHelmApp(helmApp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert release to app")
	}

	foundApp := &apptypes.App{ID: chartApp.ID, Slug: chartApp.Slug, Name: chartApp.Name}
	return foundApp, nil
}
