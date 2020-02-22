package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/license"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/session"
)

type SyncLicenseRequest struct {
	LicenseData string `json:"licenseData"`
}

type SyncLicenseResponse struct {
	ID              string                `json:"id"`
	ExpiresAt       time.Time             `json:"expiresAt"`
	ChannelName     string                `json:"channelName"`
	LicenseSequence int64                 `json:"licenseSequence"`
	LicenseType     string                `json:"licenseType"`
	Entitlements    []EntitlementResponse `json:"entitlements"`
}

type EntitlementResponse struct {
	Title string      `json:"title"`
	Value interface{} `json:"value"`
	Label string      `json:"label"`
}

func SyncLicense(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	syncLicenseRequest := SyncLicenseRequest{}
	if err := json.NewDecoder(r.Body).Decode(&syncLicenseRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		w.WriteHeader(401)
		return
	}

	foundApp, err := app.GetFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	latestLicense, err := license.Sync(foundApp, syncLicenseRequest.LicenseData)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	syncLicenseResponse := SyncLicenseResponse{
		ID:              latestLicense.Spec.LicenseID,
		ChannelName:     latestLicense.Spec.ChannelName,
		LicenseSequence: latestLicense.Spec.LicenseSequence,
		LicenseType:     latestLicense.Spec.LicenseType,
		Entitlements:    []EntitlementResponse{},
	}

	for key, entititlement := range latestLicense.Spec.Entitlements {
		if key == "expires_at" {
			if entititlement.Value.StrVal == "" {
				continue
			}

			expiration, err := time.Parse(time.RFC3339, entititlement.Value.StrVal)
			if err != nil {
				logger.Error(err)
				w.WriteHeader(500)
				return
			}
			syncLicenseResponse.ExpiresAt = expiration
		} else if key == "gitops_enabled" {
			/* do nothing */
		} else {
			syncLicenseResponse.Entitlements = append(syncLicenseResponse.Entitlements,
				EntitlementResponse{
					Title: entititlement.Title,
					Label: key,
					Value: entititlement.Value.Value(),
				})
		}
	}

	JSON(w, 200, syncLicenseResponse)
}
