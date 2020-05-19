package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/license"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/online"
	"github.com/replicatedhq/kots/kotsadm/pkg/registry"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
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

type UploadLicenseRequest struct {
	LicenseData string `json:"licenseData"`
}

type UploadLicenseResponse struct {
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
	HasPreflight   bool   `json:"hasPreflight"`
	Slug           string `json:"slug"`
	IsAirgap       bool   `json:"isAirgap"`
	NeedsRegistry  bool   `json:"needsRegistry"`
	IsConfigurable bool   `json:"isConfigurable"`
}

type ResumeInstallOnlineRequest struct {
	Slug string `json:"slug"`
}

type ResumeInstallOnlineResponse struct {
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
	HasPreflight   bool   `json:"hasPreflight"`
	Slug           string `json:"slug"`
	IsConfigurable bool   `json:"isConfigurable"`
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

func UploadNewLicense(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	uploadLicenseRequest := UploadLicenseRequest{}
	if err := json.NewDecoder(r.Body).Decode(&uploadLicenseRequest); err != nil {
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

	// validate the license
	unverifiedLicense, unsignedLicense, err := kotsutil.LoadLicenseFromBytes([]byte(uploadLicenseRequest.LicenseData))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(400)
		return
	}

	if unverifiedLicense != nil {
		uploadSignedLicense(w, uploadLicenseRequest, unverifiedLicense)
		return
	}

	if unsignedLicense != nil {
		uploadUnsignedLicense(w, uploadLicenseRequest, unsignedLicense)
		return
	}

}

func uploadUnsignedLicense(w http.ResponseWriter, uploadLicenseRequest UploadLicenseRequest, unsignedLicense *kotsv1beta1.UnsignedLicense) {
	uploadLicenseResponse := UploadLicenseResponse{
		Success: false,
	}

	desiredAppName := strings.Replace(unsignedLicense.Spec.Slug, "-", " ", 0)

	a, err := app.Create(desiredAppName, unsignedLicense.Spec.Endpoint, uploadLicenseRequest.LicenseData, false)
	if err != nil {
		logger.Error(err)
		uploadLicenseResponse.Error = err.Error()
		JSON(w, 500, uploadLicenseResponse)
		return
	}

	pendingApp := online.PendingApp{
		ID:          a.ID,
		Slug:        a.Slug,
		Name:        a.Name,
		LicenseData: uploadLicenseRequest.LicenseData,
	}
	kotsKinds, err := online.CreateAppFromOnline(&pendingApp, unsignedLicense.Spec.Endpoint)
	if err != nil {
		logger.Error(err)
		uploadLicenseResponse.Error = err.Error()
		JSON(w, 500, uploadLicenseResponse)
		return
	}

	uploadLicenseResponse.IsAirgap = false
	uploadLicenseResponse.HasPreflight = kotsKinds.Preflight != nil
	uploadLicenseResponse.Success = true
	uploadLicenseResponse.Slug = a.Slug
	uploadLicenseResponse.NeedsRegistry = false
	uploadLicenseResponse.IsConfigurable = kotsKinds.Config != nil

	JSON(w, 200, uploadLicenseResponse)
	return
}

func uploadSignedLicense(w http.ResponseWriter, uploadLicenseRequest UploadLicenseRequest, unverifiedLicense *kotsv1beta1.License) {
	uploadLicenseResponse := UploadLicenseResponse{
		Success: false,
	}

	verifiedLicense, err := kotspull.VerifySignature(unverifiedLicense)
	if err != nil {
		uploadLicenseResponse.Error = "License signature is not valid"
		JSON(w, 200, uploadLicenseResponse)
		return
	}

	desiredAppName := strings.Replace(verifiedLicense.Spec.AppSlug, "-", " ", 0)
	upstreamURI := fmt.Sprintf("replicated://%s", verifiedLicense.Spec.AppSlug)

	a, err := app.Create(desiredAppName, upstreamURI, uploadLicenseRequest.LicenseData, verifiedLicense.Spec.IsAirgapSupported)
	if err != nil {
		logger.Error(err)
		uploadLicenseResponse.Error = err.Error()
		JSON(w, 500, uploadLicenseResponse)
		return
	}

	if !verifiedLicense.Spec.IsAirgapSupported {
		// complete the install online
		pendingApp := online.PendingApp{
			ID:          a.ID,
			Slug:        a.Slug,
			Name:        a.Name,
			LicenseData: uploadLicenseRequest.LicenseData,
		}
		kotsKinds, err := online.CreateAppFromOnline(&pendingApp, upstreamURI)
		if err != nil {
			logger.Error(err)
			uploadLicenseResponse.Error = err.Error()
			JSON(w, 500, uploadLicenseResponse)
			return
		}

		uploadLicenseResponse.IsAirgap = false
		uploadLicenseResponse.HasPreflight = kotsKinds.Preflight != nil
		uploadLicenseResponse.Success = true
		uploadLicenseResponse.Slug = a.Slug
		uploadLicenseResponse.NeedsRegistry = false
		uploadLicenseResponse.IsConfigurable = kotsKinds.Config != nil

		JSON(w, 200, uploadLicenseResponse)
		return
	}

	uploadLicenseResponse.Success = true
	uploadLicenseResponse.IsAirgap = true
	uploadLicenseResponse.Slug = a.Slug

	// This is the comment from the typescript implementation \
	// and i thought it should remain

	// Carefully now, peek at registry credentials to see if we need to prompt for them
	hasKurlRegistry, err := registry.HasKurlRegistry()
	if err != nil {
		logger.Error(err)
		uploadLicenseResponse.Error = err.Error()
		JSON(w, 300, uploadLicenseRequest)
		return
	}
	uploadLicenseResponse.NeedsRegistry = !hasKurlRegistry

	JSON(w, 200, uploadLicenseResponse)
}

func ResumeInstallOnline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	resumeInstallOnlineResponse := ResumeInstallOnlineResponse{
		Success: false,
	}

	resumeInstallOnlineRequest := ResumeInstallOnlineRequest{}
	if err := json.NewDecoder(r.Body).Decode(&resumeInstallOnlineRequest); err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, 500, resumeInstallOnlineResponse)
		return
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, 500, resumeInstallOnlineResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		resumeInstallOnlineResponse.Error = "Unauthorized"
		JSON(w, 401, resumeInstallOnlineResponse)
		return
	}

	a, err := app.GetFromSlug(resumeInstallOnlineRequest.Slug)
	if err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, 500, resumeInstallOnlineResponse)
		return
	}

	pendingApp := online.PendingApp{
		ID:   a.ID,
		Slug: a.Slug,
		Name: a.Name,
	}

	// the license data is left in the table
	licenseData, err := app.GetLicenseDataFromDatabase(a.ID)
	if err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, 500, resumeInstallOnlineResponse)
		return
	}

	pendingApp.LicenseData = licenseData

	kotsLicense, _, err := kotsutil.LoadLicenseFromBytes([]byte(licenseData))
	if err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, 500, resumeInstallOnlineResponse)
		return
	}

	kotsKinds, err := online.CreateAppFromOnline(&pendingApp, fmt.Sprintf("replicated://%s", kotsLicense.Spec.AppSlug))
	if err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, 500, resumeInstallOnlineResponse)
		return
	}

	resumeInstallOnlineResponse.HasPreflight = kotsKinds.Preflight != nil
	resumeInstallOnlineResponse.Success = true
	resumeInstallOnlineResponse.Slug = a.Slug
	resumeInstallOnlineResponse.IsConfigurable = kotsKinds.Config != nil

	JSON(w, 200, resumeInstallOnlineResponse)
}
