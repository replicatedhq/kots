package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/airgap"
	"github.com/replicatedhq/kots/pkg/automation"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/tasks"
	"github.com/replicatedhq/kots/pkg/update"
	"github.com/replicatedhq/kots/pkg/util"
)

type CreateAppFromAirgapRequest struct {
	RegistryHost string `json:"registryHost"`
	Namespace    string `json:"namespace"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	IsReadOnly   bool   `json:"isReadOnly"`
}
type CreateAppFromAirgapResponse struct {
}

type UpdateAppFromAirgapRequest struct {
	AppID string `json:"appId"`
}
type UpdateAppFromAirgapResponse struct {
}

type AirgapBundleProgressResponse struct {
	Progress float64 `json:"progress"`
}

type AirgapBundleExistsResponse struct {
	Exists bool `json:"exists"`
}

type GetAirgapUploadConfigResponse struct {
	SimultaneousUploads int `json:"simultaneousUploads"`
}

var uploadedAirgapBundleChunks = map[string]struct{}{}
var chunkLock sync.Mutex
var fileLock sync.Mutex

func (h *Handler) GetAirgapInstallStatus(w http.ResponseWriter, r *http.Request) {
	appID, err := store.GetStore().GetAppIDFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to app for slug %s", mux.Vars(r)["appSlug"]))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	status, err := store.GetStore().GetAirgapInstallStatus(appID)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get install status for app %s", mux.Vars(r)["appSlug"]))
		w.WriteHeader(500)
		return
	}

	JSON(w, 200, status)
}

func (h *Handler) ResetAirgapInstallStatus(w http.ResponseWriter, r *http.Request) {
	appID, err := store.GetStore().GetAppIDFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = store.GetStore().ResetAirgapInstallInProgress(appID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetAirgapUploadConfig(w http.ResponseWriter, r *http.Request) {
	response := GetAirgapUploadConfigResponse{}

	i, _ := strconv.Atoi(os.Getenv("AIRGAP_UPLOAD_PARALLELISM"))
	if i > 0 {
		response.SimultaneousUploads = i
	} else {
		response.SimultaneousUploads = 3
	}

	JSON(w, http.StatusOK, response)
}

func (h *Handler) CheckAirgapBundleChunk(w http.ResponseWriter, r *http.Request) {
	resumableIdentifier := r.FormValue("resumableIdentifier")
	resumableChunkNumber := r.FormValue("resumableChunkNumber")
	resumableTotalChunks := r.FormValue("resumableTotalChunks")

	if resumableIdentifier == "" || resumableChunkNumber == "" {
		logger.Error(errors.New("missing resumable upload parameters"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	chunkNumber, err := strconv.ParseInt(resumableChunkNumber, 10, 64)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse chunk number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	totalChunks, err := strconv.ParseInt(resumableTotalChunks, 10, 64)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse total chunks number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	chunkKey := getChunkKey(resumableIdentifier, chunkNumber)
	if !isChunkPresent(chunkKey) {
		w.WriteHeader(http.StatusNoContent) // instead of 404 to avoid unwarranted notices in browser consoles
		return
	}

	if chunkNumber%25 == 0 {
		logger.Infof("checking chunk %d / %d. chunk key: %s", chunkNumber, totalChunks, chunkKey)
	}

	JSON(w, http.StatusOK, "")
}

func (h *Handler) UploadAirgapBundleChunk(w http.ResponseWriter, r *http.Request) {
	resumableIdentifier := r.FormValue("resumableIdentifier")
	resumableTotalChunks := r.FormValue("resumableTotalChunks")
	resumableTotalSize := r.FormValue("resumableTotalSize")
	resumableChunkNumber := r.FormValue("resumableChunkNumber")
	resumableChunkSize := r.FormValue("resumableChunkSize")

	totalChunks, err := strconv.ParseInt(resumableTotalChunks, 10, 64)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse total chunks number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	totalSize, err := strconv.ParseInt(resumableTotalSize, 10, 64)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse total size as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	chunkNumber, err := strconv.ParseInt(resumableChunkNumber, 10, 64)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse chunk number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	chunkSize, err := strconv.ParseInt(resumableChunkSize, 10, 64)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse chunk size as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// read chunk data
	airgapBundleChunk, _, err := r.FormFile("file")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer airgapBundleChunk.Close()

	airgapBundlePath := getAirgapBundlePath(resumableIdentifier)

	func() {
		// create airgap bundle file if not exists
		fileLock.Lock()
		defer fileLock.Unlock()

		_, err = os.Stat(airgapBundlePath)
		if os.IsNotExist(err) {

			// this is a new upload.  assume only one upload can happen at a time and free up some ephemeral storage.
			cleanupTempAirgapBundles()

			f, err := os.Create(airgapBundlePath)
			if err != nil {
				logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer f.Close()

			if err := f.Truncate(totalSize); err != nil {
				logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}()

	airgapBundle, err := os.OpenFile(airgapBundlePath, os.O_RDWR, 0644)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer airgapBundle.Close()

	chunkOffset := (chunkNumber - 1) * chunkSize
	if _, err := airgapBundle.Seek(chunkOffset, 0); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(airgapBundle, airgapBundleChunk); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	chunkKey := getChunkKey(resumableIdentifier, chunkNumber)
	addUploadedChank(chunkKey)

	if chunkNumber%25 == 0 {
		logger.Infof("written chunk number %d / %d. bundle id: %s", chunkNumber, totalChunks, resumableIdentifier)
	}

	// check if upload is complete
	uploadComplete := isUploadComplete(resumableIdentifier, totalChunks)
	if uploadComplete {
		logger.Infof("bundle upload complete. bundle id: %s", resumableIdentifier)
	}

	JSON(w, http.StatusOK, "")
}

func (h *Handler) AirgapBundleProgress(w http.ResponseWriter, r *http.Request) {
	identifier := mux.Vars(r)["identifier"]
	totalChunksStr := mux.Vars(r)["totalChunks"]

	totalChunks, err := strconv.ParseInt(totalChunksStr, 10, 64)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse total chunks number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	uploadProgress := getUploadProgress(identifier, totalChunks)

	airgapBundleProgressResponse := AirgapBundleProgressResponse{
		Progress: uploadProgress,
	}

	JSON(w, http.StatusOK, airgapBundleProgressResponse)
}

func (h *Handler) AirgapBundleExists(w http.ResponseWriter, r *http.Request) {
	identifier := mux.Vars(r)["identifier"]
	totalChunksStr := mux.Vars(r)["totalChunks"]

	totalChunks, err := strconv.ParseInt(totalChunksStr, 10, 64)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse total chunks number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	uploadComplete := isUploadComplete(identifier, totalChunks)

	airgapBundleExistsResponse := AirgapBundleExistsResponse{
		Exists: uploadComplete,
	}

	JSON(w, http.StatusOK, airgapBundleExistsResponse)
}

func (h *Handler) UpdateAppFromAirgap(w http.ResponseWriter, r *http.Request) {
	updateAppFromAirgapRequest := UpdateAppFromAirgapRequest{}
	if err := json.NewDecoder(r.Body).Decode(&updateAppFromAirgapRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	a, err := store.GetStore().GetApp(updateAppFromAirgapRequest.AppID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	identifier := mux.Vars(r)["identifier"]
	airgapBundlePath := getAirgapBundlePath(identifier)

	totalChunksStr := mux.Vars(r)["totalChunks"]
	totalChunks, err := strconv.ParseInt(totalChunksStr, 10, 64)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse total chunks number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// this is to avoid a race condition where the UI polls the task status before it is set by the goroutine
	if err := tasks.SetTaskStatus("update-download", "Processing...", "running"); err != nil {
		logger.Error(errors.Wrap(err, "failed to set task status"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	go func() {
		if util.IsEmbeddedCluster() {
			if err := airgap.UpdateAppFromECBundle(a.Slug, airgapBundlePath); err != nil {
				logger.Error(errors.Wrap(err, "failed to update app from ec airgap bundle"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {
			if err := airgap.UpdateAppFromAirgap(a, airgapBundlePath, false, false, false); err != nil {
				logger.Error(errors.Wrap(err, "failed to update app from airgap bundle"))
				// if NoRetry is set, we stll want to clean up immediately
				cause := errors.Cause(err)
				if err, ok := cause.(util.ActionableError); !ok || !err.NoRetry {
					return
				}
			}
		}
		if err := cleanUp(identifier, totalChunks); err != nil {
			logger.Error(errors.Wrap(err, "failed to clean up"))
		}
	}()

	updateAppFromAirgapResponse := UpdateAppFromAirgapResponse{}

	JSON(w, http.StatusAccepted, updateAppFromAirgapResponse)
}

func (h *Handler) CreateAppFromAirgap(w http.ResponseWriter, r *http.Request) {
	createAppFromAirgapRequest := CreateAppFromAirgapRequest{}
	if err := json.NewDecoder(r.Body).Decode(&createAppFromAirgapRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	pendingApp, err := store.GetStore().GetPendingAirgapUploadApp()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get k8s clientset"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var namespace string
	var isReadOnly bool
	registryHost, username, password := kotsutil.GetEmbeddedRegistryCreds(clientset)

	// if found embedded registry creds, use embedded registry
	if registryHost != "" {
		namespace = pendingApp.Slug
	} else {
		registryHost = createAppFromAirgapRequest.RegistryHost
		namespace = createAppFromAirgapRequest.Namespace
		username = createAppFromAirgapRequest.Username
		password = createAppFromAirgapRequest.Password
		isReadOnly = createAppFromAirgapRequest.IsReadOnly
	}

	identifier := mux.Vars(r)["identifier"]
	airgapBundlePath := getAirgapBundlePath(identifier)

	totalChunksStr := mux.Vars(r)["totalChunks"]
	totalChunks, err := strconv.ParseInt(totalChunksStr, 10, 64)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse total chunks number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	go func() {
		createAppOpts := airgap.CreateAirgapAppOpts{
			PendingApp:         pendingApp,
			AirgapBundle:       airgapBundlePath,
			RegistryHost:       registryHost,
			RegistryNamespace:  namespace,
			RegistryUsername:   username,
			RegistryPassword:   password,
			RegistryIsReadOnly: isReadOnly,
		}
		if err := airgap.CreateAppFromAirgap(createAppOpts); err != nil {
			logger.Error(errors.Wrap(err, "failed to create app from airgap bundle"))

			// if NoRetry is set, we stll want to clean up immediately
			cause := errors.Cause(err)
			if err, ok := cause.(util.ActionableError); !ok || !err.NoRetry {
				return
			}
		}

		if err := cleanUp(identifier, totalChunks); err != nil {
			logger.Error(errors.Wrap(err, "failed to clean up"))
		}
	}()

	createAppFromAirgapResponse := CreateAppFromAirgapResponse{}

	JSON(w, http.StatusAccepted, createAppFromAirgapResponse)
}

func getChunkKey(uploadedFileIdentifier string, chunkNumber int64) string {
	return fmt.Sprintf("%s_part_%d", uploadedFileIdentifier, chunkNumber)
}

func getAirgapBundlePath(uploadedFileIdentifier string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s.chunks.airgap", uploadedFileIdentifier))
}

func cleanupTempAirgapBundles() {
	glob := filepath.Join(os.TempDir(), "*.chunks.airgap")
	files, err := filepath.Glob(glob)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list temp airgap bundles"))
		return
	}

	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to delete temp airgap bundle"))
		}
	}
}

func addUploadedChank(chunkKey string) {
	chunkLock.Lock()
	defer chunkLock.Unlock()
	uploadedAirgapBundleChunks[chunkKey] = struct{}{}
}

func isChunkPresent(chunkKey string) bool {
	chunkLock.Lock()
	defer chunkLock.Unlock()
	_, ok := uploadedAirgapBundleChunks[chunkKey]
	return ok
}

func getUploadProgress(uploadedFileIdentifier string, totalChunks int64) float64 {
	chunkLock.Lock()
	defer chunkLock.Unlock()

	var numOfUploadedChunks int64 = 0

	var i int64
	for i = 1; i <= totalChunks; i++ {
		chunkKey := getChunkKey(uploadedFileIdentifier, i)
		if _, ok := uploadedAirgapBundleChunks[chunkKey]; ok {
			numOfUploadedChunks++
		}
	}

	return float64(numOfUploadedChunks) / float64(totalChunks)
}

func isUploadComplete(uploadedFileIdentifier string, totalChunks int64) bool {
	chunkLock.Lock()
	defer chunkLock.Unlock()

	isUploadComplete := true

	var i int64
	for i = 1; i <= totalChunks; i++ {
		chunkKey := getChunkKey(uploadedFileIdentifier, i)
		if _, ok := uploadedAirgapBundleChunks[chunkKey]; !ok {
			isUploadComplete = false
		}
	}

	return isUploadComplete
}

func cleanUp(uploadedFileIdentifier string, totalChunks int64) error {
	chunkLock.Lock()
	defer chunkLock.Unlock()

	var i int64
	for i = 1; i <= totalChunks; i++ {
		chunkKey := getChunkKey(uploadedFileIdentifier, i)
		delete(uploadedAirgapBundleChunks, chunkKey)
	}

	airgapBundlePath := getAirgapBundlePath(uploadedFileIdentifier)
	if err := os.RemoveAll(airgapBundlePath); err != nil {
		return errors.Wrap(err, "failed to remove airgap bundle")
	}

	return nil
}

func (h *Handler) UploadInitialAirgapApp(w http.ResponseWriter, r *http.Request) {
	if err := requireValidKOTSToken(w, r); err != nil {
		logger.Error(errors.Wrap(err, "failed to validate token"))
		return
	}

	appSlug := r.FormValue("appSlug")
	archiveFile, archiveHeader, err := r.FormFile("appArchive")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get form file reader"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer archiveFile.Close()

	appArchive, err := ioutil.ReadAll(archiveFile)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get read form file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	opts := automation.AutomateInstallOptions{
		AppSlug:         appSlug,
		AdditionalFiles: map[string][]byte{archiveHeader.Filename: appArchive},
	}
	err = automation.AutomateInstall(opts)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to install app"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) UploadAirgapUpdate(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]

	app, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get app for slug %q", appSlug))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	contentType := strings.Split(r.Header.Get("Content-Type"), ";")[0]
	contentType = strings.TrimSpace(contentType)

	if contentType != "multipart/form-data" {
		logger.Error(errors.Errorf("unsupported content type: %s", r.Header.Get("Content-Type")))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !app.IsAirgap {
		logger.Error(errors.New("not an airgap app"))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Cannot update an online install using an airgap update"))
		return
	}

	formReader, err := r.MultipartReader()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get multipart reader"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	foundAirgapUpdate := false
	for {
		part, err := formReader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}
			logger.Error(errors.Wrap(err, "failed to get next part"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if part.FormName() != "application.airgap" {
			continue
		}

		foundAirgapUpdate = true

		tmpFile, err := os.CreateTemp("", "kots-airgap")
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to create temp file"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(tmpFile.Name())

		_, err = io.Copy(tmpFile, part)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to copy part data"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := update.RegisterAirgapUpdate(app.Slug, tmpFile.Name()); err != nil {
			logger.Error(errors.Wrap(err, "failed to registry airgap update"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if !foundAirgapUpdate {
		logger.Error(errors.New("no airgap update found in form data"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	JSON(w, http.StatusOK, struct{}{})
}
