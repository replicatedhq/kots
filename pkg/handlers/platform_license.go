package handlers

import (
	"encoding/json"
	"net/http"
	"os"

	license "github.com/replicatedhq/kots/pkg/kotsadmlicense"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
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

	endpoint := os.Getenv("REPLICATED_API_ENDPOINT")
	if endpoint == "" {
		endpoint = util.DefaultReplicatedAPIEndpoint()
	}
	kotsLicenseData, err := license.GetFromPlatformLicense(endpoint, request.LicenseData)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	response := ExchangePlatformLicenseResponse{LicenseData: kotsLicenseData}
	JSON(w, 200, response)
}
