package handlers

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/replicatedhq/kots/kotsadm/pkg/license"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
)

const (
	DefaultAPIEndpoint = "https://replicated.app"
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

	apiEndpoint := os.Getenv("REPLICATED_API_ENDPOINT")
	if apiEndpoint == "" {
		apiEndpoint = DefaultAPIEndpoint
	}
	kotsLicenseData, err := license.GetFromPlatformLicense(apiEndpoint, request.LicenseData)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	response := ExchangePlatformLicenseResponse{LicenseData: kotsLicenseData}
	JSON(w, 200, response)
}
