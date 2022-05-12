package handlers

import (
	"net/http"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
)

// IsHelmManagedResponse - response body for the is helm managed endpoint
type IsHelmManagedResponse struct {
	Success       bool `json:"success"`
	IsHelmManaged bool `json:"isHelmManaged"`
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
