package handlers

import (
	"encoding/json"
	"net/http"

	license "github.com/replicatedhq/kots/pkg/kotsadmlicense"
	"github.com/replicatedhq/kots/pkg/logger"
)

type ExchangePlatformLicenseRequest struct {
	LicenseData string `json:"licenseData"`
}

type ExchangePlatformLicenseResponse struct {
	LicenseData string `json:"licenseData"`
}

func (h *Handler) ExchangePlatformLicense(w http.ResponseWriter, r *http.Request) {
	request := ExchangePlatformLicenseRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.Error(err)
		w.WriteHeader(400)
		return
	}

	kotsLicenseData, err := license.GetFromPlatformLicense(getReplicatedAPIEndpoint(), request.LicenseData)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	response := ExchangePlatformLicenseResponse{LicenseData: kotsLicenseData}
	JSON(w, 200, response)
}
