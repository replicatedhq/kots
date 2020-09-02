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
	Namespace string `json:"namespace"`
	Username string `json:"username"`
	Password string `json:"password"`
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
	resumableFilename := r.FormValue("resumableFilename")
	resumableChunkNumber := r.FormValue("resumableChunkNumber")

	if resumableIdentifier == "" || resumableFilename == "" || resumableChunkNumber == "" {
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

	chunksDir := filepath.Join(os.TempDir(), resumableIdentifier)
	chunkName := getChunkName(resumableFilename, chunkNumber)
	chunkPath := filepath.Join(chunksDir, chunkName)

	logger.Infof("getting airgap bundle chunk %s", chunkPath)

	_, err = os.Stat(chunkPath)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, "")
}

func UploadAirgapBundleChunk(w http.ResponseWriter, r *http.Request) {
	resumableIdentifier := r.FormValue("resumableIdentifier")
	resumableTotalChunks := r.FormValue("resumableTotalChunks")
	resumableChunkNumber := r.FormValue("resumableChunkNumber")
	resumableFilename := r.FormValue("resumableFilename")

	totalChunks, err := strconv.ParseInt(resumableTotalChunks, 10, 64)
	if err != nil {
		logger.Error(errors.New("failed to parse total chunks number as integer"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	chunkNumber, err := strconv.ParseInt(resumableChunkNumber, 10, 64)
	if err != nil {
		logger.Error(errors.New("failed to parse chunk number as integer"))
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

	// create chunks dir if not exists
	chunksDir := filepath.Join(os.TempDir(), resumableIdentifier)
	_, err = os.Stat(chunksDir)
	if os.IsNotExist(err) {
		err := os.Mkdir(chunksDir, 0644)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// create chunk file and write chunk data to it
	chunkName := getChunkName(resumableFilename, chunkNumber)
	chunkPath := filepath.Join(chunksDir, chunkName)
	chunkFile, err := os.OpenFile(chunkPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
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
	if _, err = chunkFile.Write(buf.Bytes()); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	chunkFile.Close()

	logger.Infof("saved airgap bundle chunk %s, total chunks %d", chunkPath, totalChunks)

	// check if upload is complete
	uploadComplete, err := isUploadComplete(resumableFilename, chunksDir, totalChunks)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if uploadComplete {
		airgapBundlePath := getAirgapBundlePath(resumableIdentifier)
		if err := createAirgapBundleFromChunks(airgapBundlePath, resumableFilename, chunksDir, totalChunks); err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		logger.Infof("airgap bundle saved to: %s", airgapBundlePath)
	}

	JSON(w, http.StatusOK, "")
}

func AirgapBundleExists(w http.ResponseWriter, r *http.Request) {
	airgapBundlePath := getAirgapBundlePath(mux.Vars(r)["identifier"])
	airgapBundleExists := false

	_, err := os.Stat(airgapBundlePath)
	if err != nil {
		if os.IsNotExist(err) {
			airgapBundleExists = false
		} else {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		airgapBundleExists = true
	}

	airgapBundleExistsResponse := AirgapBundleExistsResponse{
		Exists: airgapBundleExists,
	}

	JSON(w, http.StatusOK, airgapBundleExistsResponse)
}

func ProcessAirgapBundle(w http.ResponseWriter, r *http.Request) {
	airgapBundlePath := getAirgapBundlePath(mux.Vars(r)["identifier"])

	if r.Method == "POST" {
		createAppFromAirgap(w, r, airgapBundlePath)
		return
	}

	updateAppFromAirgap(w, r, airgapBundlePath)
}

func updateAppFromAirgap(w http.ResponseWriter, r *http.Request, airgapBundlePath string) {
	updateAppFromAirgapRequest := UpdateAppFromAirgapRequest{}
	if err := json.NewDecoder(r.Body).Decode(&updateAppFromAirgapRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	a, err := store.GetStore().GetApp(updateAppFromAirgapRequest.AppID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	go func() {
		if err := airgap.UpdateAppFromAirgap(a, airgapBundlePath); err != nil {
			logger.Error(errors.Wrap(err, "failed to update app from airgap bundle"))
			return
		}
		// app updated successfully, we can remove the airgap bundle
		if err := os.RemoveAll(airgapBundlePath); err != nil {
			logger.Error(errors.Wrap(err, "failed to remove airgap bundle after update"))
		}
	}()

	updateAppFromAirgapResponse := UpdateAppFromAirgapResponse{}

	JSON(w, 202, updateAppFromAirgapResponse)
}

func createAppFromAirgap(w http.ResponseWriter, r *http.Request, airgapBundlePath string) {
	createAppFromAirgapRequest := CreateAppFromAirgapRequest{}
	if err := json.NewDecoder(r.Body).Decode(&createAppFromAirgapRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
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

	go func() {
		if err := airgap.CreateAppFromAirgap(pendingApp, airgapBundlePath, registryHost, namespace, username, password); err != nil {
			logger.Error(errors.Wrap(err, "failed to create app from airgap bundle"))
			return
		}
		// app created successfully, we can remove the airgap bundle
		if err := os.RemoveAll(airgapBundlePath); err != nil {
			logger.Error(errors.Wrap(err, "failed to remove airgap bundle after create"))
		}
	}()

	createAppFromAirgapResponse := CreateAppFromAirgapResponse{}

	JSON(w, http.StatusOK, createAppFromAirgapResponse)
}

func getChunkName(uploadedFileName string, chunkNumber int64) string {
	return fmt.Sprintf("%s_part_%d", uploadedFileName, chunkNumber)
}

func getAirgapBundlePath(uploadedFileIdentifier string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s.%s", uploadedFileIdentifier, "airgap"))
}

func isUploadComplete(uploadedFileName string, chunksDir string, totalChunks int64) (bool, error) {
	isUploadComplete := true

	var i int64
	for i = 1; i <= totalChunks; i++ {
		chunkName := getChunkName(uploadedFileName, i)
		chunkPath := filepath.Join(chunksDir, chunkName)
		_, err := os.Stat(chunkPath)
		if os.IsNotExist(err) {
			isUploadComplete = false
		} else if err != nil {
			return false, errors.Wrap(err, "failed to os state file")
		}
	}

	return isUploadComplete, nil
}

func createAirgapBundleFromChunks(airgapBundlePath string, uploadedFileName string, chunksDir string, totalChunks int64) error {
	airgapBundle, err := os.OpenFile(airgapBundlePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open target file")
	}
	defer airgapBundle.Close()

	var i int64
	for i = 1; i <= totalChunks; i++ {
		// get chunk data
		chunkName := getChunkName(uploadedFileName, i)
		chunkPath := filepath.Join(chunksDir, chunkName)
		chunkFile, err := os.OpenFile(chunkPath, os.O_RDONLY, 0644)
		if err != nil {
			return errors.Wrap(err, "failed to open chunk file")
		}

		// write chunk data to target file
		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, chunkFile); err != nil {
			return errors.Wrap(err, "failed to copy chunk data to buffer")
		}
		if _, err = airgapBundle.Write(buf.Bytes()); err != nil {
			return errors.Wrap(err, "failed to write chunk buffer to target file")
		}

		chunkFile.Close()
	}

	// airgap file was created successfully from chunks, we can remove the chunks dir
	if err := os.RemoveAll(chunksDir); err != nil {
		logger.Error(errors.Wrap(err, "failed to remove chunks directory"))
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
