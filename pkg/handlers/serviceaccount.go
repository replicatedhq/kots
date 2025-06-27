package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	kotsadmlicense "github.com/replicatedhq/kots/pkg/kotsadmlicense"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
)

type ServiceAccountToken struct {
	Identity string `json:"i"`
	Secret   string `json:"s"`
}

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

	saToken, err := validateServiceAccountToken(request.ServiceAccountToken, currentLicense.Spec.LicenseID)
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

func validateServiceAccountToken(token, currentLicenseID string) (*ServiceAccountToken, error) {
	tokenBytes, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode service account token")
	}

	var saToken ServiceAccountToken
	if err := json.Unmarshal(tokenBytes, &saToken); err != nil {
		return nil, errors.Wrap(err, "failed to parse service account token")
	}

	if saToken.Identity == "" {
		return nil, errors.New("service account token missing identity")
	}

	if saToken.Secret == "" {
		return nil, errors.New("service account token missing secret")
	}

	currentIdentity, err := extractIdentityFromLicenseID(currentLicenseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract current license identity")
	}

	if saToken.Identity != currentIdentity {
		return nil, errors.New("Identity mismatch: token identity does not match current license identity")
	}

	return &saToken, nil
}

func extractIdentityFromLicenseID(licenseID string) (string, error) {
	if decoded, err := base64.StdEncoding.DecodeString(licenseID); err == nil {
		var token ServiceAccountToken
		if err := json.Unmarshal(decoded, &token); err == nil {
			return token.Identity, nil
		}
	}

	return licenseID, nil
}