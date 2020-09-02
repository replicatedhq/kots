package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/airgap"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type CreateAppFromAirgapRequest struct {
	RegistryHost string `json:"registryHost"`
	Namespace    string `json:"namespace"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}
type CreateAppFromAirgapResponse struct {
}

type UpdateAppFromAirgapRequest struct {
	AppID string `json:"appId"`
}
type UpdateAppFromAirgapResponse struct {
}

type AirgapBundleExistsResponse struct {
	Exists bool `json:"exists"`
}

var uploadedAirgapBundleChunks = map[string]struct{}{}

func GetAirgapInstallStatus(w http.ResponseWriter, r *http.Request) {
	status, err := store.GetStore().GetAirgapInstallStatus()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	JSON(w, 200, status)
}

func ResetAirgapInstallStatus(w http.ResponseWriter, r *http.Request) {
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

func CheckAirgapBundleChunk(w http.ResponseWriter, r *http.Request) {
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
		logger.Error(errors.New("failed to parse chunk number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	totalChunks, err := strconv.ParseInt(resumableTotalChunks, 10, 64)
	if err != nil {
		logger.Error(errors.New("failed to parse total chunks number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	chunkKey := getChunkKey(resumableIdentifier, chunkNumber)
	if _, ok := uploadedAirgapBundleChunks[chunkKey]; !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if chunkNumber % 25 == 0 {
		logger.Infof("checking chunk %d / %d. chunk key: %s", chunkNumber, totalChunks, chunkKey)
	}

	JSON(w, http.StatusOK, "")
}

func UploadAirgapBundleChunk(w http.ResponseWriter, r *http.Request) {
	resumableIdentifier := r.FormValue("resumableIdentifier")
	resumableTotalChunks := r.FormValue("resumableTotalChunks")
	resumableTotalSize := r.FormValue("resumableTotalSize")
	resumableChunkNumber := r.FormValue("resumableChunkNumber")
	resumableChunkSize := r.FormValue("resumableChunkSize")

	totalChunks, err := strconv.ParseInt(resumableTotalChunks, 10, 64)
	if err != nil {
		logger.Error(errors.New("failed to parse total chunks number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	totalSize, err := strconv.ParseInt(resumableTotalSize, 10, 64)
	if err != nil {
		logger.Error(errors.New("failed to parse total size as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	chunkNumber, err := strconv.ParseInt(resumableChunkNumber, 10, 64)
	if err != nil {
		logger.Error(errors.New("failed to parse chunk number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	chunkSize, err := strconv.ParseInt(resumableChunkSize, 10, 64)
	if err != nil {
		logger.Error(errors.New("failed to parse chunk size as integer"))
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

	// create airgap bundle file if not exists
	airgapBundlePath := getAirgapBundlePath(resumableIdentifier)
	_, err = os.Stat(airgapBundlePath)
	if os.IsNotExist(err) {
		f, err := os.Create(airgapBundlePath)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
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

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, airgapBundleChunk); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err = airgapBundle.Write(buf.Bytes()); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	chunkKey := getChunkKey(resumableIdentifier, chunkNumber)
	uploadedAirgapBundleChunks[chunkKey] = struct{}{}

	if chunkNumber % 25 == 0 {
		logger.Infof("written chunk number %d / %d. bundle id: %s", chunkNumber, totalChunks, resumableIdentifier)
	}

	// check if upload is complete
	uploadComplete := isUploadComplete(resumableIdentifier, totalChunks)
	if uploadComplete {
		logger.Infof("bundle upload complete. bundle id: %s", resumableIdentifier)
	}

	JSON(w, http.StatusOK, "")
}

func AirgapBundleExists(w http.ResponseWriter, r *http.Request) {
	identifier := mux.Vars(r)["identifier"]
	totalChunksStr := mux.Vars(r)["totalChunks"]

	totalChunks, err := strconv.ParseInt(totalChunksStr, 10, 64)
	if err != nil {
		logger.Error(errors.New("failed to parse total chunks number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	uploadComplete := isUploadComplete(identifier, totalChunks)

	airgapBundleExistsResponse := AirgapBundleExistsResponse{
		Exists: uploadComplete,
	}

	JSON(w, http.StatusOK, airgapBundleExistsResponse)
}

func ProcessAirgapBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		createAppFromAirgap(w, r)
		return
	}

	updateAppFromAirgap(w, r)
}

func updateAppFromAirgap(w http.ResponseWriter, r *http.Request) {
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
		logger.Error(errors.New("failed to parse total chunks number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	go func() {
		if err := airgap.UpdateAppFromAirgap(a, airgapBundlePath); err != nil {
			logger.Error(errors.Wrap(err, "failed to update app from airgap bundle"))
			return
		}
		// app updated successfully, we can remove the airgap bundle
		if err := cleanUp(identifier, totalChunks); err != nil {
			logger.Error(errors.Wrap(err, "failed to clean up"))
		}
	}()

	updateAppFromAirgapResponse := UpdateAppFromAirgapResponse{}

	JSON(w, 202, updateAppFromAirgapResponse)
}

func createAppFromAirgap(w http.ResponseWriter, r *http.Request) {
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

	var registryHost, namespace, username, password string
	registryHost, username, password, err = getKurlRegistryCreds()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// if found kurl registry creds, use kurl registry
	if registryHost != "" {
		namespace = pendingApp.Slug
	} else {
		registryHost = createAppFromAirgapRequest.RegistryHost
		namespace = createAppFromAirgapRequest.Namespace
		username = createAppFromAirgapRequest.Username
		password = createAppFromAirgapRequest.Password
	}

	identifier := mux.Vars(r)["identifier"]
	airgapBundlePath := getAirgapBundlePath(identifier)

	totalChunksStr := mux.Vars(r)["totalChunks"]
	totalChunks, err := strconv.ParseInt(totalChunksStr, 10, 64)
	if err != nil {
		logger.Error(errors.New("failed to parse total chunks number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	go func() {
		if err := airgap.CreateAppFromAirgap(pendingApp, airgapBundlePath, registryHost, namespace, username, password); err != nil {
			logger.Error(errors.Wrap(err, "failed to create app from airgap bundle"))
			return
		}
		// app created successfully, we can remove the airgap bundle
		if err := cleanUp(identifier, totalChunks); err != nil {
			logger.Error(errors.Wrap(err, "failed to clean up"))
		}
	}()

	createAppFromAirgapResponse := CreateAppFromAirgapResponse{}

	JSON(w, http.StatusOK, createAppFromAirgapResponse)
}

func getChunkKey(uploadedFileIdentifier string, chunkNumber int64) string {
	return fmt.Sprintf("%s_part_%d", uploadedFileIdentifier, chunkNumber)
}

func getAirgapBundlePath(uploadedFileIdentifier string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s.%s", uploadedFileIdentifier, "airgap"))
}

func isUploadComplete(uploadedFileIdentifier string, totalChunks int64) bool {
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
	var i int64
	for i = 1; i <= totalChunks; i++ {
		chunkKey := getChunkKey(uploadedFileIdentifier, i)
		delete(uploadedAirgapBundleChunks, chunkKey);
	}

	airgapBundlePath := getAirgapBundlePath(uploadedFileIdentifier)
	if err := os.RemoveAll(airgapBundlePath); err != nil {
		return errors.Wrap(err, "failed to remove airgap bundle")
	}

	return nil
}

func getKurlRegistryCreds() (hostname string, username string, password string, finalErr error) {
	cfg, err := config.GetConfig()
	if err != nil {
		finalErr = errors.Wrap(err, "failed to get cluster config")
		return
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		finalErr = errors.Wrap(err, "failed to create kubernetes clientset")
		return
	}

	// kURL registry secret is always in default namespace
	secret, err := clientset.CoreV1().Secrets("default").Get(context.TODO(), "registry-creds", metav1.GetOptions{})
	if err != nil {
		return
	}

	dockerJson, ok := secret.Data[".dockerconfigjson"]
	if !ok {
		return
	}

	type dockerRegistryAuth struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Auth     string `json:"auth"`
	}
	dockerConfig := struct {
		Auths map[string]dockerRegistryAuth `json:"auths"`
	}{}

	err = json.Unmarshal(dockerJson, &dockerConfig)
	if err != nil {
		return
	}

	for host, auth := range dockerConfig.Auths {
		if auth.Username == "kurl" {
			hostname = host
			username = auth.Username
			password = auth.Password
			return
		}
	}

	return
}
