package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmlicense "github.com/replicatedhq/kots/pkg/kotsadmlicense"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/online"
	installationtypes "github.com/replicatedhq/kots/pkg/online/types"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/updatechecker"
	updatecheckertypes "github.com/replicatedhq/kots/pkg/updatechecker/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

type SyncLicenseRequest struct {
	LicenseData string `json:"licenseData"`
}

type LicenseResponse struct {
	ID                             string                `json:"id"`
	Assignee                       string                `json:"assignee"`
	ExpiresAt                      time.Time             `json:"expiresAt"`
	ChannelName                    string                `json:"channelName"`
	LicenseSequence                int64                 `json:"licenseSequence"`
	LicenseType                    string                `json:"licenseType"`
	Entitlements                   []EntitlementResponse `json:"entitlements"`
	IsAirgapSupported              bool                  `json:"isAirgapSupported"`
	IsGitOpsSupported              bool                  `json:"isGitOpsSupported"`
	IsIdentityServiceSupported     bool                  `json:"isIdentityServiceSupported"`
	IsGeoaxisSupported             bool                  `json:"isGeoaxisSupported"`
	IsSemverRequired               bool                  `json:"isSemverRequired"`
	IsSnapshotSupported            bool                  `json:"isSnapshotSupported"`
	IsDisasterRecoverySupported    bool                  `json:"isDisasterRecoverySupported"`
	LastSyncedAt                   string                `json:"lastSyncedAt"`
	IsSupportBundleUploadSupported bool                  `json:"isSupportBundleUploadSupported"`
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

	appSlug := mux.Vars(r)["appSlug"]

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		syncLicenseResponse.Error = "failed to get app from slug"
		logger.Error(errors.Wrap(err, syncLicenseResponse.Error))
		JSON(w, http.StatusInternalServerError, syncLicenseResponse)
		return
	}

	currentLicense, err := store.GetStore().GetLatestLicenseForApp(foundApp.ID)
	if err != nil {
		syncLicenseResponse.Error = "failed to get current license"
		logger.Error(errors.Wrap(err, syncLicenseResponse.Error))
		JSON(w, http.StatusInternalServerError, syncLicenseResponse)
		return
	}

	latestLicense, isSynced, err := kotsadmlicense.Sync(foundApp, syncLicenseRequest.LicenseData, true)
	if err != nil {
		syncLicenseResponse.Error = "failed to sync license"
		logger.Error(errors.Wrap(err, syncLicenseResponse.Error))
		JSON(w, http.StatusInternalServerError, syncLicenseResponse)
		return
	}

	if !foundApp.IsAirgap && currentLicense.Spec.ChannelID != latestLicense.Spec.ChannelID {
		// channel changed and this is an online installation, fetch the latest release for the new channel
		go func(appID string) {
			opts := updatecheckertypes.CheckForUpdatesOpts{
				AppID: appID,
			}
			_, err := updatechecker.CheckForUpdates(opts)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to fetch the latest release for the new channel"))
			}
		}(foundApp.ID)
	}

	licenseResponse, err := licenseResponseFromLicense(latestLicense, foundApp)
	if err != nil {
		syncLicenseResponse.Error = "failed to get license response from license"
		logger.Error(errors.Wrap(err, syncLicenseResponse.Error))
		JSON(w, http.StatusInternalServerError, syncLicenseResponse)
		return
	}

	syncLicenseResponse.Success = true
	syncLicenseResponse.Synced = isSynced
	syncLicenseResponse.License = *licenseResponse

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

	licenseResponse, err := licenseResponseFromLicense(license, foundApp)
	if err != nil {
		getLicenseResponse.Error = "failed to get license response from license"
		logger.Error(errors.Wrap(err, getLicenseResponse.Error))
		JSON(w, http.StatusInternalServerError, getLicenseResponse)
		return
	}

	getLicenseResponse.Success = true
	getLicenseResponse.License = *licenseResponse

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
		logger.Error(errors.Wrap(err, "failed to decode request body"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	licenseString := uploadLicenseRequest.LicenseData

	// validate the license
	unverifiedLicense, err := kotsutil.LoadLicenseFromBytes([]byte(licenseString))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to load license from bytes"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	uploadLicenseResponse := UploadLicenseResponse{
		Success: false,
	}

	verifiedLicense, err := kotslicense.VerifySignature(unverifiedLicense)
	if err != nil {
		uploadLicenseResponse.Error = "License signature is not valid"
		if _, ok := err.(kotslicense.LicenseDataError); ok {
			uploadLicenseResponse.Error = fmt.Sprintf("%s: %s", uploadLicenseResponse.Error, err.Error())
		}
		JSON(w, http.StatusBadRequest, uploadLicenseResponse)
		return
	}

	if !kotsadm.IsAirgap() {
		// sync license
		logger.Info("syncing license with server to retrieve latest version")
		licenseData, err := replicatedapp.GetLatestLicense(verifiedLicense)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get latest license"))
			uploadLicenseResponse.Error = err.Error()
			JSON(w, http.StatusInternalServerError, uploadLicenseResponse)
			return
		}
		verifiedLicense = licenseData.License
		licenseString = string(licenseData.LicenseBytes)
	}

	// check license expiration
	expired, err := kotslicense.LicenseIsExpired(verifiedLicense)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to check if license is expired"))
		uploadLicenseResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, uploadLicenseResponse)
		return
	}
	if expired {
		uploadLicenseResponse.Error = "License is expired"
		JSON(w, http.StatusBadRequest, uploadLicenseResponse)
		return
	}

	// check if license already exists
	existingLicense, err := kotsadmlicense.CheckIfLicenseExists([]byte(licenseString))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to check if license already exists"))
		uploadLicenseResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, uploadLicenseResponse)
		return
	}

	if existingLicense != nil {
		resolved, err := kotsadmlicense.ResolveExistingLicense(verifiedLicense)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to resolve existing license conflict"))
		}

		if !resolved {
			uploadLicenseResponse.Error = "License already exists"
			uploadLicenseResponse.DeleteAppCommand = fmt.Sprintf("kubectl kots remove %s -n %s --force", existingLicense.Spec.AppSlug, util.PodNamespace)
			JSON(w, http.StatusBadRequest, uploadLicenseResponse)
			return
		}
	}

	installationParams, err := kotsutil.GetInstallationParams(kotsadmtypes.KotsadmConfigMap)
	if err != nil {
		logger.Error(err)
		uploadLicenseResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, uploadLicenseResponse)
		return
	}

	desiredAppName := strings.Replace(verifiedLicense.Spec.AppSlug, "-", " ", 0)
	upstreamURI := fmt.Sprintf("replicated://%s", verifiedLicense.Spec.AppSlug)

	// TODO: FLORIAN - Validate requested channel ID here and back fill as needed!
	a, err := store.GetStore().CreateApp(desiredAppName, installationParams.RequestedChannelID, upstreamURI, licenseString, verifiedLicense.Spec.IsAirgapSupported, installationParams.SkipImagePush, installationParams.RegistryIsReadOnly)
	if err != nil {
		logger.Error(err)
		uploadLicenseResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, uploadLicenseResponse)
		return
	}

	if !verifiedLicense.Spec.IsAirgapSupported || util.IsEmbeddedCluster() {
		// complete the install online
		createAppOpts := online.CreateOnlineAppOpts{
			PendingApp: &installationtypes.PendingApp{
				ID:           a.ID,
				Slug:         a.Slug,
				Name:         a.Name,
				LicenseData:  uploadLicenseRequest.LicenseData,
				VersionLabel: installationParams.AppVersionLabel,
			},
			UpstreamURI: upstreamURI,
		}
		kotsKinds, err := online.CreateAppFromOnline(createAppOpts)
		if err != nil {
			logger.Error(err)
			uploadLicenseResponse.Error = err.Error()
			JSON(w, http.StatusInternalServerError, uploadLicenseResponse)
			return
		}

		err = kotsutil.RemoveAppVersionLabelFromInstallationParams(kotsadmtypes.KotsadmConfigMap)
		if err != nil {
			logger.Error(err)
			uploadLicenseResponse.Error = err.Error()
			JSON(w, http.StatusInternalServerError, uploadLicenseResponse)
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

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		uploadLicenseResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, uploadLicenseRequest)
		return
	}
	uploadLicenseResponse.NeedsRegistry = !kotsutil.HasEmbeddedRegistry(clientset)

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
		JSON(w, http.StatusInternalServerError, resumeInstallOnlineResponse)
		return
	}

	a, err := store.GetStore().GetAppFromSlug(resumeInstallOnlineRequest.Slug)
	if err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, resumeInstallOnlineResponse)
		return
	}

	installationParams, err := kotsutil.GetInstallationParams(kotsadmtypes.KotsadmConfigMap)
	if err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, resumeInstallOnlineResponse)
		return
	}

	pendingApp := installationtypes.PendingApp{
		ID:           a.ID,
		Slug:         a.Slug,
		Name:         a.Name,
		VersionLabel: installationParams.AppVersionLabel,
	}

	// the license data is left in the table
	kotsLicense, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, resumeInstallOnlineResponse)
		return
	}

	// marshal it
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(kotsLicense, &b); err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, resumeInstallOnlineResponse)
		return
	}
	pendingApp.LicenseData = string(b.Bytes())

	createAppOpts := online.CreateOnlineAppOpts{
		PendingApp:  &pendingApp,
		UpstreamURI: fmt.Sprintf("replicated://%s", kotsLicense.Spec.AppSlug),
	}
	kotsKinds, err := online.CreateAppFromOnline(createAppOpts)
	if err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, resumeInstallOnlineResponse)
		return
	}

	err = kotsutil.RemoveAppVersionLabelFromInstallationParams(kotsadmtypes.KotsadmConfigMap)
	if err != nil {
		logger.Error(err)
		resumeInstallOnlineResponse.Error = err.Error()
		JSON(w, http.StatusInternalServerError, resumeInstallOnlineResponse)
		return
	}

	resumeInstallOnlineResponse.HasPreflight = kotsKinds.HasPreflights()
	resumeInstallOnlineResponse.Success = true
	resumeInstallOnlineResponse.Slug = a.Slug
	resumeInstallOnlineResponse.IsConfigurable = kotsKinds.IsConfigurable()

	JSON(w, http.StatusOK, resumeInstallOnlineResponse)
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

type ChangeLicenseRequest struct {
	LicenseData string `json:"licenseData"`
}

type ChangeLicenseResponse struct {
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	License LicenseResponse `json:"license"`
}

func (h *Handler) ChangeLicense(w http.ResponseWriter, r *http.Request) {
	changeLicenseResponse := ChangeLicenseResponse{
		Success: false,
	}

	changeLicenseRequest := ChangeLicenseRequest{}
	if err := json.NewDecoder(r.Body).Decode(&changeLicenseRequest); err != nil {
		errMsg := "failed to decode request body"
		logger.Error(errors.Wrap(err, errMsg))
		changeLicenseResponse.Error = errMsg
		JSON(w, http.StatusBadRequest, changeLicenseResponse)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]
	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		errMsg := "failed to get app from slug"
		logger.Error(errors.Wrap(err, errMsg))
		changeLicenseResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, changeLicenseResponse)
		return
	}

	currentLicense, err := store.GetStore().GetLatestLicenseForApp(foundApp.ID)
	if err != nil {
		errMsg := "failed to get current license"
		logger.Error(errors.Wrap(err, errMsg))
		changeLicenseResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, changeLicenseResponse)
		return
	}

	newLicense, err := kotsadmlicense.Change(foundApp, changeLicenseRequest.LicenseData)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to change license"))
		changeLicenseResponse.Error = errors.Cause(err).Error()
		JSON(w, http.StatusInternalServerError, changeLicenseResponse)
		return
	}

	if !foundApp.IsAirgap && currentLicense.Spec.ChannelID != newLicense.Spec.ChannelID {
		// channel changed and this is an online installation, fetch the latest release for the new channel
		go func(appID string) {
			opts := updatecheckertypes.CheckForUpdatesOpts{
				AppID: appID,
			}
			_, err := updatechecker.CheckForUpdates(opts)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to fetch the latest release for the new channel"))
			}
		}(foundApp.ID)
	}

	licenseResponse, err := licenseResponseFromLicense(newLicense, foundApp)
	if err != nil {
		errMsg := "failed to get license response from license"
		logger.Error(errors.Wrap(err, errMsg))
		changeLicenseResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, changeLicenseResponse)
		return
	}

	changeLicenseResponse.Success = true
	changeLicenseResponse.License = *licenseResponse

	JSON(w, 200, changeLicenseResponse)
}

func licenseResponseFromLicense(license *kotsv1beta1.License, app *apptypes.App) (*LicenseResponse, error) {
	entitlements, expiresAt, err := getLicenseEntitlements(license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get license entitlements")
	}

	sort.Slice(entitlements, func(i, j int) bool {
		return entitlements[i].Title < entitlements[j].Title
	})

	response := LicenseResponse{
		ID:                             license.Spec.LicenseID,
		Assignee:                       license.Spec.CustomerName,
		ChannelName:                    license.Spec.ChannelName,
		LicenseSequence:                license.Spec.LicenseSequence,
		LicenseType:                    license.Spec.LicenseType,
		Entitlements:                   entitlements,
		ExpiresAt:                      expiresAt,
		IsAirgapSupported:              license.Spec.IsAirgapSupported,
		IsGitOpsSupported:              license.Spec.IsGitOpsSupported,
		IsIdentityServiceSupported:     license.Spec.IsIdentityServiceSupported,
		IsGeoaxisSupported:             license.Spec.IsGeoaxisSupported,
		IsSemverRequired:               license.Spec.IsSemverRequired,
		IsSnapshotSupported:            license.Spec.IsSnapshotSupported,
		IsDisasterRecoverySupported:    license.Spec.IsDisasterRecoverySupported,
		LastSyncedAt:                   app.LastLicenseSync,
		IsSupportBundleUploadSupported: license.Spec.IsSupportBundleUploadSupported,
	}

	return &response, nil
}
