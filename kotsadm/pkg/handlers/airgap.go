package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/airgap"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type CreateAppFromAirgapRequest struct {
}

type CreateAppFromAirgapResponse struct {
}

type UpdateAppFromAirgapResponse struct {
}

func GetAirgapInstallStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	status, err := airgap.GetInstallStatus()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	JSON(w, 200, status)
}

func UploadAirgapBundle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	if r.Method == "POST" {
		createAppFromAirgap(w, r)
		return
	}

	updateAppFromAirgap(w, r)
}

func updateAppFromAirgap(w http.ResponseWriter, r *http.Request) {
	a, err := app.Get(r.FormValue("appId"))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	airgapBundle, _, err := r.FormFile("file")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	go func() {
		defer airgapBundle.Close()
		if err := airgap.UpdateAppFromAirgap(a, airgapBundle); err != nil {
			logger.Error(err)
		}
	}()

	updateAppFromAirgapResponse := UpdateAppFromAirgapResponse{}

	JSON(w, 202, updateAppFromAirgapResponse)
}

func createAppFromAirgap(w http.ResponseWriter, r *http.Request) {
	pendingApp, err := airgap.GetPendingAirgapUploadApp()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	var registryHost, namespace, username, password string
	registryHost, username, password, err = getKurlRegistryCreds()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// if found kurl registry creds, use kurl registry
	if registryHost != "" {
		namespace = pendingApp.Slug
	} else {
		registryHost = r.FormValue("registryHost")
		namespace = r.FormValue("namespace")
		username = r.FormValue("username")
		password = r.FormValue("password")
	}

	airgapBundle, _, err := r.FormFile("file")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	go func() {
		defer airgapBundle.Close()
		if err := airgap.CreateAppFromAirgap(pendingApp, airgapBundle, registryHost, namespace, username, password); err != nil {
			logger.Error(err)
		}
	}()

	createAppFromAirgapResponse := CreateAppFromAirgapResponse{}

	JSON(w, 202, createAppFromAirgapResponse)
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
