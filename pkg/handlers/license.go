package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	license "github.com/replicatedhq/kots/pkg/kotsadmlicense"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/online"
	installationtypes "github.com/replicatedhq/kots/pkg/online/types"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/registry"
	"github.com/replicatedhq/kots/pkg/store"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

type SyncLicenseRequest struct {
	LicenseData string `json:"licenseData"`
}

type LicenseResponse struct {
	ID                         string                `json:"id"`
	Assignee                   string                `json:"assignee"`
	ExpiresAt                  time.Time             `json:"expiresAt"`
	ChannelName                string                `json:"channelName"`
	LicenseSequence            int64                 `json:"licenseSequence"`
	LicenseType                string                `json:"licenseType"`
	Entitlements               []EntitlementResponse `json:"entitlements"`
	IsAirgapSupported          bool                  `json:"isAirgapSupported"`
	IsGitOpsSupported          bool                  `json:"isGitOpsSupported"`
	IsIdentityServiceSupported bool                  `json:"isIdentityServiceSupported"`
	IsGeoaxisSupported         bool                  `json:"isGeoaxisSupported"`
	IsSnapshotSupported        bool                  `json:"isSnapshotSupported"`
}

type SyncLicenseResponse struct {
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	Synced  bool            `json:"synced"`
	License LicenseResponse `json:"license"`
}

type GetLicenseResponse struct {
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	License LicenseResponse `json:"license"`
}

type EntitlementResponse struct {
	Title     string      `json:"title"`
	Value     interface{} `json:"value"`
	Label     string      `json:"label"`
	ValueType string      `json:"valueType"`
}

type UploadLicenseRequest struct {
	LicenseData string `json:"licenseData"`
}

type UploadLicenseResponse struct {
	Success          bool   `json:"success"`
	Error            string `json:"error,omitempty"`
	DeleteAppCommand string `json:"deleteAppCommand,omitempty"`
	HasPreflight     bool   `json:"hasPreflight"`
	Slug             string `json:"slug"`
	IsAirgap         bool   `json:"isAirgap"`
	NeedsRegistry    bool   `json:"needsRegistry"`
	IsConfigurable   bool   `json:"isConfigurable"`
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

type GetOnlineInstallStatusErrorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) SyncLicense(w http.ResponseWriter, r *http.Request) {
	syncLicenseResponse := SyncLicenseResponse{
		Success: false,
	}

	syncLicenseRequest := SyncLicenseRequest{}
	if err := json.NewDecoder(r.Body).Decode(&syncLicenseRequest); err != nil {
		syncLicenseResponse.Error = "failed to decode request"
		logger.Error(errors.Wrap(err, syncLicenseResponse.Error))
		JSON(w, http.StatusInternalServerError, syncLicenseResponse)
		return
	}

	foundApp, err := store.GetStore().GetAppFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		syncLicenseResponse.Error = "failed to get app from slug"
		logger.Error(errors.Wrap(err, syncLicenseResponse.Error))
		JSON(w, http.StatusInternalServerError, syncLicenseResponse)
		return
	}

	latestLicense, synced, err := license.Sync(foundApp, syncLicenseRequest.LicenseData, true)
	if err != nil {
		syncLicenseResponse.Error = "failed to sync license"
		logger.Error(errors.Wrap(err, syncLicenseResponse.Error))
		JSON(w, http.StatusInternalServerError, syncLicenseResponse)
		return
	}

	entitlements, expiresAt, err := getLicenseEntitlements(latestLicense)
	if err != nil {
		syncLicenseResponse.Error = "failed to get license entitlements"
		logger.Error(errors.Wrap(err, syncLicenseResponse.Error))
		JSON(w, http.StatusInternalServerError, syncLicenseResponse)
		return
	}

	syncLicenseResponse.Success = true
	syncLicenseResponse.Synced = synced
	syncLicenseResponse.License = LicenseResponse{
		ID:                         latestLicense.Spec.LicenseID,
		Assignee:                   latestLicense.Spec.CustomerName,
		ChannelName:                latestLicense.Spec.ChannelName,
		LicenseSequence:            latestLicense.Spec.LicenseSequence,
		LicenseType:                latestLicense.Spec.LicenseType,
		Entitlements:               entitlements,
		ExpiresAt:                  expiresAt,
		IsAirgapSupported:          latestLicense.Spec.IsAirgapSupported,
		IsGitOpsSupported:          latestLicense.Spec.IsGitOpsSupported,
		IsIdentityServiceSupported: latestLicense.Spec.IsIdentityServiceSupported,
		IsGeoaxisSupported:         latestLicense.Spec.IsGeoaxisSupported,
		IsSnapshotSupported:        latestLicense.Spec.IsSnapshotSupported,
	}

	JSON(w, http.StatusOK, syncLicenseResponse)
}

func (h *Handler) GetLicense(w http.ResponseWriter, r *http.Request) {
	getLicenseResponse := GetLicenseResponse{
		Success: false,
	}

	appSlug := mux.Vars(r)["appSlug"]
	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		getLicenseResponse.Error = "failed to get app from slug"
		logger.Error(errors.Wrap(err, getLicenseResponse.Error))
		JSON(w, http.StatusInternalServerError, getLicenseResponse)
		return
	}

	license, err := store.GetStore().GetLatestLicenseForApp(foundApp.ID)
	if err != nil {
		getLicenseResponse.Error = "failed to get license for app"
		logger.Error(errors.Wrap(err, getLicenseResponse.Error))
		JSON(w, http.StatusInternalServerError, getLicenseResponse)
		return
	}

	entitlements, expiresAt, err := getLicenseEntitlements(license)
	if err != nil {
		getLicenseResponse.Error = "failed to get license entitlements"
		logger.Error(errors.Wrap(err, getLicenseResponse.Error))
		JSON(w, http.StatusInternalServerError, getLicenseResponse)
		return
	}

	getLicenseResponse.Success = true
	getLicenseResponse.License = LicenseResponse{
		ID:                         license.Spec.LicenseID,
		Assignee:                   license.Spec.CustomerName,
		ChannelName:                license.Spec.ChannelName,
		LicenseSequence:            license.Spec.LicenseSequence,
		LicenseType:                license.Spec.LicenseType,
		Entitlements:               entitlements,
		ExpiresAt:                  expiresAt,
		IsAirgapSupported:          license.Spec.IsAirgapSupported,
		IsGitOpsSupported:          license.Spec.IsGitOpsSupported,
		IsIdentityServiceSupported: license.Spec.IsIdentityServiceSupported,
		IsGeoaxisSupported:         license.Spec.IsGeoaxisSupported,
		IsSnapshotSupported:        license.Spec.IsSnapshotSupported,
	}

	JSON(w, http.StatusOK, getLicenseResponse)
}

func getLicenseEntitlements(license *kotsv1beta1.License) ([]EntitlementResponse, time.Time, error) {
	var expiresAt time.Time
	entitlements := []EntitlementResponse{}

	for key, entititlement := range license.Spec.Entitlements {
		if key == "expires_at" {
			if entititlement.Value.StrVal == "" {
				continue
			}

			expiration, err := time.Parse(time.RFC3339, entititlement.Value.StrVal)
			if err != nil {
				return nil, time.Time{}, errors.Wrap(err, "failed to parse expiration date")
			}
			expiresAt = expiration
		} else if key == "gitops_enabled" {
			/* do nothing */
		} else if !entititlement.IsHidden {
			entitlements = append(entitlements,
				EntitlementResponse{
					Title:     entititlement.Title,
					Label:     key,
					Value:     entititlement.Value.Value(),
					ValueType: entititlement.ValueType,
				})
		}
	}

	return entitlements, expiresAt, nil
}

func (h *Handler) UploadNewLicense(w http.ResponseWriter, r *http.Request) {
	uploadLicenseRequest := UploadLicenseRequest{}
	if err := json.NewDecoder(r.Body).Decode(&uploadLicenseRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	licenseString := uploadLicenseRequest.LicenseData

	// validate the license
	unverifiedLicense, err := kotsutil.LoadLicenseFromBytes([]byte(licenseString))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(400)
		return
	}

	uploadLicenseResponse := UploadLicenseResponse{
		Success: false,
	}

	verifiedLicense, err := kotspull.VerifySignature(unverifiedLicense)
	if err != nil {
		uploadLicenseResponse.Error = "License signature is not valid"
		JSON(w, 400, uploadLicenseResponse)
		return
	}

	disableOutboundConnections := false
	// ignore the error, default to false
	disableOutboundConnections, _ = strconv.ParseBool(os.Getenv("DISABLE_OUTBOUND_CONNECTIONS"))
	if !disableOutboundConnections {
		// sync license
		logger.Info("syncing license with server to retrieve latest version")
		licenseData, err := kotslicense.GetLatestLicense(verifiedLicense)
		if err != nil {
			logger.Error(err)
			uploadLicenseResponse.Error = err.Error()
			JSON(w, 500, uploadLicenseResponse)
			return
		}
		verifiedLicense = licenseData.License
		licenseString = string(licenseData.LicenseBytes)
	}

	// check license expiration
	expired, err := kotspull.LicenseIsExpired(verifiedLicense)
	if err != nil {
		logger.Error(err)
		uploadLicenseResponse.Error = err.Error()
		JSON(w, 500, uploadLicenseResponse)
		return
	}
	if expired {
		uploadLicenseResponse.Error = "License is expired"
		JSON(w, 400, uploadLicenseResponse)
		return
	}

	allLicenses, err := store.GetStore().GetAllAppLicenses()
	if err != nil {
		logger.Error(err)
		uploadLicenseResponse.Error = err.Error()
		JSON(w, 500, uploadLicenseResponse)
		return
	}

	// check does license already exist
	existingLicense, err := license.CheckDoesLicenseExists(allLicenses, uploadLicenseRequest.LicenseData)
	if err != nil {
		logger.Error(err)
		uploadLicenseResponse.Error = err.Error()
		JSON(w, 500, uploadLicenseResponse)
		return
	}

	if existingLicense != nil {
		uploadLicenseResponse.Error = "License already exists"
		uploadLicenseResponse.DeleteAppCommand = fmt.Sprintf("kubectl kots remove %s -n %s --force", existingLicense.Spec.AppSlug, os.Getenv("POD_NAMESPACE"))
		JSON(w, 400, uploadLicenseResponse)
		return
	}

	installationParams, err := kotsutil.GetInstallationParams(kotsadmtypes.KotsadmConfigMap)
	if err != nil {
		logger.Error(err)
		uploadLicenseResponse.Error = err.Error()
		JSON(w, 500, uploadLicenseResponse)
		return
	}

	desiredAppName := strings.Replace(verifiedLicense.Spec.AppSlug, "-", " ", 0)
	upstreamURI := fmt.Sprintf("replicated://%s", verifiedLicense.Spec.AppSlug)

	a, err := store.GetStore().CreateApp(desiredAppName, upstreamURI, licenseString, verifiedLicense.Spec.IsAirgapSupported, installationParams.SkipImagePush, installationParams.RegistryIsReadOnly)
	if err != nil {
		logger.Error(err)
		uploadLicenseResponse.Error = err.Error()
		JSON(w, 500, uploadLicenseResponse)
		return
	}

	if !verifiedLicense.Spec.IsAirgapSupported {
		// complete the install online
		pendingApp := installationtypes.PendingApp{
			ID:          a.ID,
			Slug:        a.Slug,
			Name:        a.Name,
			LicenseData: uploadLicenseRequest.LicenseData,
		}
		kotsKinds, err := online.CreateAppFromOnline(&pendingApp, upstreamURI, false, false)
		if err != nil {
			logger.Error(err)
			uploadLicenseResponse.Error = err.Error()
			JSON(w, 500, uploadLicenseResponse)
			return
		}

		uploadLicenseResponse.HasPreflight = kotsKinds.HasPreflights()
		uploadLicenseResponse.IsAirgap = false
		uploadLicenseResponse.Success = true
		uploadLicenseResponse.Slug = a.Slug
		uploadLicenseResponse.NeedsRegistry = false
		uploadLicenseResponse.IsConfigurable = kotsKinds.IsConfigurable()

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

func (h *Handler) ResumeInstallOnline(w http.ResponseWriter, r *http.Request) {
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

	a, err := store.GetStore().GetAppFromSlug(resumeInstallOnlineRequest.Slug)
	if err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, 500, resumeInstallOnlineResponse)
		return
	}

	pendingApp := installationtypes.PendingApp{
		ID:   a.ID,
		Slug: a.Slug,
		Name: a.Name,
	}

	// the license data is left in the table
	kotsLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, 500, resumeInstallOnlineResponse)
		return
	}

	// marshal it
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(kotsLicense, &b); err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, 500, resumeInstallOnlineResponse)
		return
	}

	pendingApp.LicenseData = string(b.Bytes())

	kotsKinds, err := online.CreateAppFromOnline(&pendingApp, fmt.Sprintf("replicated://%s", kotsLicense.Spec.AppSlug), false, false)
	if err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, 500, resumeInstallOnlineResponse)
		return
	}

	resumeInstallOnlineResponse.HasPreflight = kotsKinds.HasPreflights()
	resumeInstallOnlineResponse.Success = true
	resumeInstallOnlineResponse.Slug = a.Slug
	resumeInstallOnlineResponse.IsConfigurable = kotsKinds.IsConfigurable()

	JSON(w, 200, resumeInstallOnlineResponse)
}

func (h *Handler) GetOnlineInstallStatus(w http.ResponseWriter, r *http.Request) {
	status, err := store.GetStore().GetPendingInstallationStatus()
	if err != nil {
		logger.Error(err)
		JSON(w, 500, GetOnlineInstallStatusErrorResponse{
			Error: fmt.Sprintf("failed to get install status: %v", err),
		})
		return
	}

	JSON(w, 200, status)
}

// GetPlatformLicenseCompatibility route is UNAUTHENTICATED
// Authentication must be added here which will break backwards compatibility.
// This route exists for backwards compatibility with platform License API and should be called by
// the application only.
func (h *Handler) GetPlatformLicenseCompatibility(w http.ResponseWriter, r *http.Request) {
	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		if store.GetStore().IsNotFound(err) {
			JSON(w, http.StatusNotFound, struct{}{})
			return
		}
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(apps) == 0 {
		JSON(w, http.StatusNotFound, struct{}{})
		return
	}

	if len(apps) > 1 {
		JSON(w, http.StatusBadRequest, struct{}{})
		return
	}

	app := apps[0]
	license, err := store.GetStore().GetLatestLicenseForApp(app.ID)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, struct{}{})
		return
	}

	type licenseFieldType struct {
		Field            string      `json:"field"`
		Title            string      `json:"title"`
		Type             string      `json:"type"`
		Value            interface{} `json:"value"`
		HideFromCustomer bool        `json:"hide_from_customer,omitempty"`
	}

	type licenseType struct {
		LicenseID      string             `json:"license_id"`
		InstallationID string             `json:"installation_id"`
		Assignee       string             `json:"assignee"`
		ReleaseChannel string             `json:"release_channel"`
		LicenseType    string             `json:"license_type"`
		ExpirationTime string             `json:"expiration_time,omitempty"`
		Fields         []licenseFieldType `json:"fields"`
	}

	platformLicense := licenseType{
		LicenseID:      license.Spec.LicenseID,
		InstallationID: app.ID,
		Assignee:       license.Spec.CustomerName,
		ReleaseChannel: license.Spec.ChannelName,
		LicenseType:    license.Spec.LicenseType,
		Fields:         make([]licenseFieldType, 0),
	}

	for k, e := range license.Spec.Entitlements {
		if k == "expires_at" {
			if e.Value.StrVal != "" {
				platformLicense.ExpirationTime = e.Value.StrVal
			}
			continue
		}

		field := licenseFieldType{
			Field:            k,
			Title:            e.Title,
			Type:             e.ValueType,
			Value:            e.Value.Value(),
			HideFromCustomer: e.IsHidden,
		}

		platformLicense.Fields = append(platformLicense.Fields, field)
	}

	JSON(w, http.StatusOK, platformLicense)
	return
}
