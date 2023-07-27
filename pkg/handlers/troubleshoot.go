package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/helm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	"github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/pkg/util"
	redact2 "github.com/replicatedhq/troubleshoot/pkg/redact"
	tsupportbundle "github.com/replicatedhq/troubleshoot/pkg/supportbundle"
	tsupportbundletypes "github.com/replicatedhq/troubleshoot/pkg/supportbundle/types"
)

type GetSupportBundleResponse struct {
	ID         string                       `json:"id"`
	Slug       string                       `json:"slug"`
	AppID      string                       `json:"appId"`
	Name       string                       `json:"name"`
	Size       float64                      `json:"size"`
	Status     string                       `json:"status"`
	TreeIndex  string                       `json:"treeIndex"`
	CreatedAt  time.Time                    `json:"createdAt"`
	UploadedAt *time.Time                   `json:"uploadedAt"`
	UpdatedAt  *time.Time                   `json:"updatedAt"`
	SharedAt   *time.Time                   `json:"sharedAt"`
	IsArchived bool                         `json:"isArchived"`
	Analysis   *types.SupportBundleAnalysis `json:"analysis"`
	Progress   *types.SupportBundleProgress `json:"progress"`
}

type GetSupportBundleFilesResponse struct {
	Files map[string][]byte `json:"files"`

	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type ListSupportBundlesResponse struct {
	SupportBundles []ResponseSupportBundle `json:"supportBundles"`
}
type ResponseSupportBundle struct {
	ID         string                       `json:"id"`
	Slug       string                       `json:"slug"`
	AppID      string                       `json:"appId"`
	Name       string                       `json:"name"`
	Size       float64                      `json:"size"`
	Status     string                       `json:"status"`
	CreatedAt  time.Time                    `json:"createdAt"`
	UploadedAt *time.Time                   `json:"uploadedAt"`
	SharedAt   *time.Time                   `json:"sharedAt"`
	IsArchived bool                         `json:"isArchived"`
	Analysis   *types.SupportBundleAnalysis `json:"analysis"`
}

type GetSupportBundleCommandRequest struct {
	Origin string `json:"origin"`
}

type GetSupportBundleCommandResponse struct {
	Command []string `json:"command"`
}

type CollectSupportBundlesResponse struct {
	ID    string `json:"id"`
	Slug  string `json:"slug"`
	AppID string `json:"appId"`
}

type GetSupportBundleRedactionsResponse struct {
	Redactions redact2.RedactionList `json:"redactions"`

	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type GetPodDetailsFromSupportBundleResponse struct {
	tsupportbundletypes.PodDetails `json:",inline"`

	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type PodContainer struct {
	Name            string `json:"name"`
	LogsFilePath    string `json:"logsFilePath"`
	IsInitContainer bool   `json:"isInitContainer"`
}

type PutSupportBundleRedactions struct {
	Redactions redact2.RedactionList `json:"redactions"`
}

func (h *Handler) GetSupportBundle(w http.ResponseWriter, r *http.Request) {
	bundleSlug := mux.Vars(r)["bundleSlug"]

	bundle, err := store.GetStore().GetSupportBundle(bundleSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	analysis, err := store.GetStore().GetSupportBundleAnalysis(bundle.ID)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get analysis for bundle %s", bundle.Slug))
	}

	getSupportBundleResponse := GetSupportBundleResponse{
		ID:         bundle.ID,
		Slug:       bundle.Slug,
		AppID:      bundle.AppID,
		Name:       bundle.Name,
		Size:       bundle.Size,
		Status:     string(bundle.Status),
		TreeIndex:  bundle.TreeIndex,
		CreatedAt:  bundle.CreatedAt,
		UpdatedAt:  bundle.UpdatedAt,
		UploadedAt: bundle.UploadedAt,
		SharedAt:   bundle.SharedAt,
		IsArchived: bundle.IsArchived,
		Analysis:   analysis,
		Progress:   &bundle.Progress,
	}

	JSON(w, http.StatusOK, getSupportBundleResponse)
}

func (h *Handler) GetSupportBundleFiles(w http.ResponseWriter, r *http.Request) {
	getSupportBundleFilesResponse := GetSupportBundleFilesResponse{
		Success: false,
	}

	bundleID := mux.Vars(r)["bundleId"]
	filenames := r.URL.Query()["filename"]

	bundleArchive, err := store.GetStore().GetSupportBundleArchive(bundleID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get support bundle archive"))
		getSupportBundleFilesResponse.Error = "failed to get support bundle archive"
		JSON(w, http.StatusInternalServerError, nil)
		return
	}
	defer os.RemoveAll(bundleArchive)

	files, err := tsupportbundle.GetFilesContents(bundleArchive, filenames)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get files"))
		getSupportBundleFilesResponse.Error = "failed to get files"
		JSON(w, http.StatusInternalServerError, getSupportBundleFilesResponse)
		return
	}

	getSupportBundleFilesResponse.Success = true
	getSupportBundleFilesResponse.Files = files

	JSON(w, http.StatusOK, getSupportBundleFilesResponse)
}

func (h *Handler) ListSupportBundles(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	appIDOrSlug := appSlug
	if !util.IsHelmManaged() {
		a, err := store.GetStore().GetAppFromSlug(appSlug)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		appIDOrSlug = a.ID
	}

	supportBundles, err := store.GetStore().ListSupportBundles(appIDOrSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	responseSupportBundles := []ResponseSupportBundle{}
	for _, bundle := range supportBundles {
		analysis, err := store.GetStore().GetSupportBundleAnalysis(bundle.ID)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to get analysis for bundle %s", bundle.Slug))
		}

		responseSupportBundle := ResponseSupportBundle{
			ID:         bundle.ID,
			Slug:       bundle.Slug,
			AppID:      bundle.AppID,
			Name:       bundle.Name,
			Size:       bundle.Size,
			Status:     string(bundle.Status),
			CreatedAt:  bundle.CreatedAt,
			UploadedAt: bundle.UploadedAt,
			IsArchived: bundle.IsArchived,
			SharedAt:   bundle.SharedAt,
			Analysis:   analysis,
		}

		responseSupportBundles = append(responseSupportBundles, responseSupportBundle)
	}

	listSupportBundlesResponse := ListSupportBundlesResponse{
		SupportBundles: responseSupportBundles,
	}

	JSON(w, http.StatusOK, listSupportBundlesResponse)
}

func (h *Handler) GetSupportBundleCommand(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	// in case of an error, return a generic command
	response := GetSupportBundleCommandResponse{}

	getSupportBundleCommandRequest := GetSupportBundleCommandRequest{}
	if err := json.NewDecoder(r.Body).Decode(&getSupportBundleCommandRequest); err != nil {
		logger.Error(errors.Wrap(err, "failed to decode request"))
		JSON(w, http.StatusOK, response)
		return
	}

	if util.IsHelmManaged() {
		helmApp := helm.GetHelmApp(appSlug)
		if helmApp == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		response.Command = []string{
			"curl https://krew.sh/support-bundle | bash",
			fmt.Sprintf("kubectl support-bundle --load-cluster-specs"),
		}

		opts := types.TroubleshootOptions{
			Origin:    getSupportBundleCommandRequest.Origin,
			InCluster: false,
		}

		if _, err := supportbundle.CreateSupportBundleDependencies(helmApp, helmApp.GetCurrentSequence(), opts); err != nil {
			logger.Error(errors.Wrap(err, "failed to create support bundle spec"))
			JSON(w, http.StatusOK, response)
			return
		}

		response.Command = supportbundle.GetBundleCommand(helmApp.GetSlug())
		JSON(w, http.StatusOK, response)
		return
	}

	response.Command = []string{
		"curl https://krew.sh/support-bundle | bash",
		fmt.Sprintf("kubectl support-bundle --load-cluster-specs"),
	}

	foundApp, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app"))
		JSON(w, http.StatusOK, response)
		return
	}

	sequence := int64(0)

	downstreams, err := store.GetStore().ListDownstreamsForApp(foundApp.ID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get downstreams for app"))
		JSON(w, http.StatusOK, response)
		return
	} else if len(downstreams) > 0 {
		currentVersion, err := store.GetStore().GetCurrentDownstreamVersion(foundApp.ID, downstreams[0].ClusterID)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get deployed app sequence"))
			JSON(w, http.StatusOK, response)
			return
		}

		if currentVersion != nil {
			sequence = currentVersion.Sequence
		}
	}

	opts := types.TroubleshootOptions{
		Origin:    getSupportBundleCommandRequest.Origin,
		InCluster: false,
	}
	if _, err := supportbundle.CreateSupportBundleDependencies(foundApp, sequence, opts); err != nil {
		logger.Error(errors.Wrap(err, "failed to create support bundle spec"))
		JSON(w, http.StatusOK, response)
		return
	}

	response.Command = supportbundle.GetBundleCommand(appSlug)

	JSON(w, http.StatusOK, response)
}

func (h *Handler) DownloadSupportBundle(w http.ResponseWriter, r *http.Request) {
	bundleID := mux.Vars(r)["bundleId"]

	bundle, err := store.GetStore().GetSupportBundle(bundleID)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, nil)
		return
	}

	bundleArchive, err := store.GetStore().GetSupportBundleArchive(bundle.ID)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, nil)
		return
	}
	defer os.RemoveAll(bundleArchive)

	f, err := os.Open(bundleArchive)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, nil)
		return
	}
	defer f.Close()

	filename := fmt.Sprintf("supportbundle-%s.tar.gz", bundle.CreatedAt.Format("2006-01-02T15_04_05"))

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, f)
}

func (h *Handler) ShareSupportBundle(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	bundleID := mux.Vars(r)["bundleId"]

	app, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, nil)
		return
	}

	if app.IsAirgap {
		logger.Error(errors.New("Support bundle sharing is not supported for airgapped installations."))
		JSON(w, http.StatusBadRequest, nil)
		return
	}

	license, err := store.GetStore().GetLatestLicenseForApp(app.ID)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, nil)
		return
	}

	if !license.Spec.IsSupportBundleUploadSupported {
		logger.Errorf("License does not have support bundle sharing enabled")
		JSON(w, http.StatusForbidden, nil)
		return
	}

	bundle, err := store.GetStore().GetSupportBundle(bundleID)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, nil)
		return
	}

	bundleArchive, err := store.GetStore().GetSupportBundleArchive(bundle.ID)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, nil)
		return
	}
	defer os.RemoveAll(bundleArchive)

	f, err := os.Open(bundleArchive)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, nil)
		return
	}
	defer f.Close()

	fileStat, err := f.Stat()
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, nil)
		return
	}

	endpoint := fmt.Sprintf("%s/supportbundle/upload/%s", license.Spec.Endpoint, license.Spec.AppSlug)

	req, err := util.NewRequest("POST", endpoint, f)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, nil)
		return
	}

	reportingInfo := reporting.GetReportingInfo(app.ID)
	reporting.InjectReportingInfoHeaders(req, reportingInfo)

	req.Header.Set("Content-Type", "application/tar+gzip")
	req.Header.Set("X-Replicated-SupportBundle-CollectedAt", bundle.CreatedAt.Format(time.RFC3339))

	req.ContentLength = fileStat.Size()

	req.SetBasicAuth(license.Spec.LicenseID, license.Spec.LicenseID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			logger.Errorf("Failed to share support bundle: %d: %s", resp.StatusCode, string(body))
		} else {
			logger.Errorf("Failed to share support bundle: %d", resp.StatusCode)
		}
		JSON(w, http.StatusInternalServerError, string(body))
		return
	}

	now := time.Now()
	bundle.SharedAt = &now
	if err := store.GetStore().UpdateSupportBundle(bundle); err != nil {
		logger.Error(errors.Wrap(err, "failed to update support bundle"))
		JSON(w, http.StatusInternalServerError, nil)
		return
	}

	JSON(w, http.StatusOK, "")
}

func (h *Handler) DeleteSupportBundle(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	bundleID := mux.Vars(r)["bundleId"]

	app, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app from slug"))
		JSON(w, http.StatusInternalServerError, nil)
		return
	}

	if err := store.GetStore().DeleteSupportBundle(bundleID, app.ID); err != nil {
		logger.Error(errors.Wrap(err, "failed to delete support bundle"))
		JSON(w, http.StatusInternalServerError, nil)
		return
	}

	JSON(w, http.StatusOK, "")
}

func (h *Handler) CollectSupportBundle(w http.ResponseWriter, r *http.Request) {
	if util.IsHelmManaged() {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	a, err := store.GetStore().GetApp(mux.Vars(r)["appId"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bundleID, err := supportbundle.Collect(a, mux.Vars(r)["clusterId"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	collectSupportBundlesResponse := CollectSupportBundlesResponse{
		ID:    bundleID,
		Slug:  bundleID,
		AppID: a.ID,
	}

	JSON(w, http.StatusAccepted, collectSupportBundlesResponse)
}

func (h *Handler) CollectHelmSupportBundle(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	if !util.IsHelmManaged() {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	helmApp := helm.GetHelmApp(appSlug)
	if helmApp == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	bundleID, err := supportbundle.CollectHelm(helmApp)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to collect helm support bundle"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	collectSupportBundlesResponse := CollectSupportBundlesResponse{
		ID:   bundleID,
		Slug: bundleID,
	}

	JSON(w, http.StatusAccepted, collectSupportBundlesResponse)
}

// UploadSupportBundle route is UNAUTHENTICATED
// This request comes from the `kubectl support-bundle` command.
func (h *Handler) UploadSupportBundle(w http.ResponseWriter, r *http.Request) {
	bundleContents, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to read request body"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tmpFile, err := os.CreateTemp("", "kots")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create temp file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpFile.Name())

	err = os.WriteFile(tmpFile.Name(), bundleContents, 0644)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to save bundle to temp file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	supportBundle, err := supportbundle.CreateBundle(mux.Vars(r)["bundleId"], mux.Vars(r)["appId"], tmpFile.Name())
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create support bundle"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// we need the app archive to get the analyzers for old support bundles that don't include the analysis in the bundle
	if err = supportbundle.CreateSupportBundleAnalysis(mux.Vars(r)["appId"], tmpFile.Name(), supportBundle); err != nil {
		logger.Error(errors.Wrap(err, "failed create analysis"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetSupportBundleRedactions(w http.ResponseWriter, r *http.Request) {
	getSupportBundleRedactionsResponse := GetSupportBundleRedactionsResponse{
		Success: false,
	}

	bundleID := mux.Vars(r)["bundleId"]
	redactions, err := store.GetStore().GetRedactions(bundleID)
	if err != nil {
		if store.GetStore().IsNotFound(err) {
			JSON(w, http.StatusNotFound, getSupportBundleRedactionsResponse)
			return
		}
		logger.Error(err)
		getSupportBundleRedactionsResponse.Error = fmt.Sprintf("failed to find redactions for bundle %s", bundleID)
		JSON(w, http.StatusInternalServerError, getSupportBundleRedactionsResponse)
		return
	}

	getSupportBundleRedactionsResponse.Success = true
	getSupportBundleRedactionsResponse.Redactions = redactions

	JSON(w, http.StatusOK, getSupportBundleRedactionsResponse)
}

func (h *Handler) GetPodDetailsFromSupportBundle(w http.ResponseWriter, r *http.Request) {
	getPodDetailsFromSupportBundleResponse := GetPodDetailsFromSupportBundleResponse{
		Success: false,
	}

	bundleID := mux.Vars(r)["bundleId"]
	podName := r.URL.Query().Get("podName")
	podNamespace := r.URL.Query().Get("podNamespace")

	bundleArchive, err := store.GetStore().GetSupportBundleArchive(bundleID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get support bundle archive"))
		JSON(w, http.StatusInternalServerError, nil)
		return
	}
	defer os.RemoveAll(bundleArchive)

	podDetails, err := tsupportbundle.GetPodDetails(bundleArchive, podNamespace, podName)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get pod details"))
		getPodDetailsFromSupportBundleResponse.Error = "failed to get pod details"
		JSON(w, http.StatusInternalServerError, getPodDetailsFromSupportBundleResponse)
		return
	}

	getPodDetailsFromSupportBundleResponse.PodDetails = *podDetails
	getPodDetailsFromSupportBundleResponse.Success = true

	JSON(w, http.StatusOK, getPodDetailsFromSupportBundleResponse)
}

// SetSupportBundleRedactions route is UNAUTHENTICATED
// This request comes from the `kubectl support-bundle` command.
func (h *Handler) SetSupportBundleRedactions(w http.ResponseWriter, r *http.Request) {
	redactionsBody, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	redactions := PutSupportBundleRedactions{}
	err = json.Unmarshal(redactionsBody, &redactions)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	bundleID := mux.Vars(r)["bundleId"]
	err = store.GetStore().SetRedactions(bundleID, redactions.Redactions)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	return
}
