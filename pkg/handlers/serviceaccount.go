package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	kotsadmlicense "github.com/replicatedhq/kots/pkg/kotsadmlicense"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
)

type ServiceAccountTokenRequest struct {
	ServiceAccountToken string `json:"serviceAccountToken"`
}

type ServiceAccountTokenResponse struct {
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	Synced  bool            `json:"synced"`
	License LicenseResponse `json:"license"`
}

func (h *Handler) UploadServiceAccountToken(w http.ResponseWriter, r *http.Request) {
	response := ServiceAccountTokenResponse{
		Success: false,
	}

	request := ServiceAccountTokenRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error = "failed to decode request"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	if request.ServiceAccountToken == "" {
		response.Error = "service account token is required"
		logger.Error(errors.New(response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		response.Error = "failed to get app from slug"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	currentLicense, err := store.GetStore().GetLatestLicenseForApp(foundApp.ID)
	if err != nil {
		response.Error = "failed to get current license"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	saToken, _, err := kotsadmlicense.ValidateServiceAccountToken(request.ServiceAccountToken, currentLicense)
	if err != nil {
		response.Error = err.Error()
		logger.Error(errors.Wrap(err, "failed to validate service account token"))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	latestLicense, isSynced, err := kotsadmlicense.SyncWithServiceAccountToken(foundApp, request.ServiceAccountToken, true)
	if err != nil {
		if errors.Cause(err) != nil && errors.Cause(err).Error() == "received 400 from upstream" {
			response.Error = "Invalid token: received 400 from upstream"
		} else {
			response.Error = "failed to sync license with service account token"
		}
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	licenseResponse, err := licenseResponseFromLicense(latestLicense, foundApp)
	if err != nil {
		response.Error = "failed to get license response from license"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	logger.Infof("Service account token uploaded successfully for app %s, identity: %s", foundApp.Slug, saToken.Identity)

	response.Success = true
	response.Synced = isSynced
	response.License = *licenseResponse

	JSON(w, http.StatusOK, response)
}
