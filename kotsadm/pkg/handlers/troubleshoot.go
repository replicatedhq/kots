package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	downstream "github.com/replicatedhq/kots/pkg/kotsadmdownstream"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/render/helper"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	"github.com/replicatedhq/kots/pkg/supportbundle/types"
	troubleshootanalyze "github.com/replicatedhq/troubleshoot/pkg/analyze"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/convert"
	redact2 "github.com/replicatedhq/troubleshoot/pkg/redact"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
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
	IsArchived bool                         `json:"isArchived"`
	Analysis   *types.SupportBundleAnalysis `json:"analysis"`
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
	IsArchived bool                         `json:"isArchived"`
	Analysis   *types.SupportBundleAnalysis `json:"analysis"`
}

type GetSupportBundleCommandRequest struct {
	Origin string `json:"origin"`
}

type GetSupportBundleCommandResponse struct {
	Command []string `json:"command"`
}

type GetSupportBundleRedactionsResponse struct {
	Redactions redact2.RedactionList `json:"redactions"`

	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type PutSupportBundleRedactions struct {
	Redactions redact2.RedactionList `json:"redactions"`
}

func (h *Handler) GetSupportBundle(w http.ResponseWriter, r *http.Request) {
	bundleSlug := mux.Vars(r)["bundleSlug"]

	bundle, err := store.GetStore().GetSupportBundleFromSlug(bundleSlug)
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
		Status:     bundle.Status,
		TreeIndex:  bundle.TreeIndex,
		CreatedAt:  bundle.CreatedAt,
		UploadedAt: bundle.UploadedAt,
		IsArchived: bundle.IsArchived,
		Analysis:   analysis,
	}

	JSON(w, http.StatusOK, getSupportBundleResponse)
}

func (h *Handler) GetSupportBundleFiles(w http.ResponseWriter, r *http.Request) {
	getSupportBundleFilesResponse := GetSupportBundleFilesResponse{
		Success: false,
	}

	bundleID := mux.Vars(r)["bundleId"]
	filenames := r.URL.Query()["filename"]

	files, err := supportbundle.GetFilesContents(bundleID, filenames)
	if err != nil {
		logger.Error(err)
		getSupportBundleFilesResponse.Error = "failed to get files"
		JSON(w, 500, getSupportBundleFilesResponse)
		return
	}

	getSupportBundleFilesResponse.Success = true
	getSupportBundleFilesResponse.Files = files

	JSON(w, http.StatusOK, getSupportBundleFilesResponse)
}

func (h *Handler) ListSupportBundles(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	supportBundles, err := store.GetStore().ListSupportBundles(a.ID)
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
			Status:     bundle.Status,
			CreatedAt:  bundle.CreatedAt,
			UploadedAt: bundle.UploadedAt,
			IsArchived: bundle.IsArchived,
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
	response := GetSupportBundleCommandResponse{
		Command: []string{
			"curl https://krew.sh/support-bundle | bash",
			fmt.Sprintf("kubectl support-bundle secret/%s/%s", os.Getenv("POD_NAMESPACE"), supportbundle.GetSpecSecretName(appSlug)),
		},
	}

	getSupportBundleCommandRequest := GetSupportBundleCommandRequest{}
	if err := json.NewDecoder(r.Body).Decode(&getSupportBundleCommandRequest); err != nil {
		logger.Error(errors.Wrap(err, "failed to decode request"))
		JSON(w, http.StatusOK, response)
		return
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
		currentVersion, err := downstream.GetCurrentVersion(foundApp.ID, downstreams[0].ClusterID)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get deployed app sequence"))
			JSON(w, http.StatusOK, response)
			return
		}

		if currentVersion != nil {
			sequence = currentVersion.Sequence
		}
	}

	if err := createSupportBundle(foundApp.ID, sequence, getSupportBundleCommandRequest.Origin, false); err != nil {
		logger.Error(errors.Wrap(err, "failed to create support bundle spec"))
		JSON(w, http.StatusOK, response)
		return
	}

	response.Command = supportbundle.GetBundleCommand(foundApp.Slug)

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

func (h *Handler) CollectSupportBundle(w http.ResponseWriter, r *http.Request) {
	a, err := store.GetStore().GetApp(mux.Vars(r)["appId"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := supportbundle.Collect(a.ID, mux.Vars(r)["clusterId"]); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusNoContent, "")
}

// UploadSupportBundle route is UNAUTHENTICATED
// This request comes from the `kubectl support-bundle` command.
func (h *Handler) UploadSupportBundle(w http.ResponseWriter, r *http.Request) {
	bundleContents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to read request body"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tmpFile, err := ioutil.TempFile("", "kots")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create temp file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpFile.Name())

	err = ioutil.WriteFile(tmpFile.Name(), bundleContents, 0644)
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

	// we need the app archive to get the analyzers
	foundApp, err := store.GetStore().GetApp(mux.Vars(r)["appId"])
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create temp dir"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(foundApp.ID, foundApp.CurrentSequence, archiveDir)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get app version archive"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to load kots kinds from archive"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	analyzer := kotsKinds.Analyzer
	// SupportBundle overwrites Analyzer if defined
	if kotsKinds.SupportBundle != nil {
		analyzer = kotsutil.SupportBundleToAnalyzer(kotsKinds.SupportBundle)
	}
	if analyzer == nil {
		analyzer = &troubleshootv1beta2.Analyzer{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "troubleshoot.sh/v1beta2",
				Kind:       "Analyzer",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "default-analyzers",
			},
			Spec: troubleshootv1beta2.AnalyzerSpec{
				Analyzers: []*troubleshootv1beta2.Analyze{},
			},
		}
	}

	if err := supportbundle.InjectDefaultAnalyzers(analyzer); err != nil {
		logger.Error(errors.Wrap(err, "failed to inject analyzers"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s := k8sjson.NewYAMLSerializer(k8sjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(analyzer, &b); err != nil {
		logger.Error(errors.Wrap(err, "failed to encode analyzers"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	renderedAnalyzers, err := helper.RenderAppFile(foundApp, nil, b.Bytes(), kotsKinds)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to render analyzers"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	analyzeResult, err := troubleshootanalyze.DownloadAndAnalyze(tmpFile.Name(), string(renderedAnalyzers))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to analyze"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := convert.FromAnalyzerResult(analyzeResult)
	insights, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to marshal result"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := store.GetStore().SetSupportBundleAnalysis(supportBundle.ID, insights); err != nil {
		logger.Error(errors.Wrap(err, "failed to save result"))
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
		logger.Error(err)
		getSupportBundleRedactionsResponse.Error = fmt.Sprintf("failed to find redactions for bundle %s", bundleID)
		JSON(w, http.StatusBadRequest, getSupportBundleRedactionsResponse)
		return
	}

	getSupportBundleRedactionsResponse.Success = true
	getSupportBundleRedactionsResponse.Redactions = redactions

	JSON(w, http.StatusOK, getSupportBundleRedactionsResponse)
}

// SetSupportBundleRedactions route is UNAUTHENTICATED
// This request comes from the `kubectl support-bundle` command.
func (h *Handler) SetSupportBundleRedactions(w http.ResponseWriter, r *http.Request) {
	redactionsBody, err := ioutil.ReadAll(r.Body)
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
