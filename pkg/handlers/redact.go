package handlers

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/session"
	v1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type UpdateRedactRequest struct {
	RedactSpec string `json:"redactSpec"`
}

type UpdateRedactResponse struct {
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	UpdatedSpec string `json:"updatedSpec"`
}

type GetRedactResponse struct {
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	UpdatedSpec string `json:"updatedSpec"`
}

func UpdateRedact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	updateRedactResponse := UpdateRedactResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		updateRedactResponse.Error = "failed to parse authorization header"
		JSON(w, 401, updateRedactResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		updateRedactResponse.Error = "failed to parse authorization header"
		JSON(w, 401, updateRedactResponse)
		return
	}

	updateRedactRequest := UpdateRedactRequest{}
	if err := json.NewDecoder(r.Body).Decode(&updateRedactRequest); err != nil {
		logger.Error(err)
		updateRedactResponse.Error = "failed to decode request body"
		JSON(w, 400, updateRedactResponse)
		return
	}

	cfg, err := config.GetConfig()
	if err != nil {
		updateRedactResponse.Error = "failed to get cluster config"
		JSON(w, 401, updateRedactResponse)
		return
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		updateRedactResponse.Error = "failed to create kubernetes clientset"
		JSON(w, 401, updateRedactResponse)
		return
	}

	configMap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get("kotsadm-redact", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			// not found, so return empty string
			updateRedactResponse.Error = errors.Wrap(err, "failed to get kotsadm-redact configMap").Error()
			JSON(w, 200, updateRedactResponse)
			return
		} else {
			// not found, so create it fresh
			newMap := v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kotsadm-redact",
					Namespace: os.Getenv("POD_NAMESPACE"),
					Labels: map[string]string{
						"kots.io/kotsadm": "true",
					},
				},
				Data: map[string]string{
					"kotsadm-redact": updateRedactRequest.RedactSpec,
				},
			}
			_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Create(&newMap)
			if err != nil {
				updateRedactResponse.Error = errors.Wrap(err, "failed to create kotsadm-redact configMap").Error()
				JSON(w, 200, updateRedactResponse)
				return
			}

			updateRedactResponse.Success = true
			updateRedactResponse.UpdatedSpec = updateRedactRequest.RedactSpec
			JSON(w, 200, updateRedactResponse)
			return
		}
	}

	configMap.Data["kotsadm-redact"] = updateRedactRequest.RedactSpec
	_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Update(configMap)
	if err != nil {
		updateRedactResponse.Error = errors.Wrap(err, "failed to update kotsadm-redact configMap").Error()
		JSON(w, 200, updateRedactResponse)
		return
	}

	updateRedactResponse.Success = true
	updateRedactResponse.UpdatedSpec = updateRedactRequest.RedactSpec
	JSON(w, 200, updateRedactResponse)
}

func GetRedact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	getRedactResponse := GetRedactResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		getRedactResponse.Error = "failed to parse authorization header"
		JSON(w, 401, getRedactResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		getRedactResponse.Error = "failed to parse authorization header"
		JSON(w, 401, getRedactResponse)
		return
	}

	cfg, err := config.GetConfig()
	if err != nil {
		getRedactResponse.Error = "failed to get cluster config"
		JSON(w, 401, getRedactResponse)
		return
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		getRedactResponse.Error = "failed to create kubernetes clientset"
		JSON(w, 401, getRedactResponse)
		return
	}

	configMap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get("kotsadm-redact", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			// not found, so return empty string
			getRedactResponse.Error = errors.Wrap(err, "failed to get kotsadm-redact configMap").Error()
			JSON(w, 200, getRedactResponse)
			return
		} else {
			// not found, so return empty string
			getRedactResponse.Success = true
			JSON(w, 200, getRedactResponse)
			return
		}
	}

	encodedData, ok := configMap.Data["kotsadm-redact"]
	if !ok {
		getRedactResponse.Error = "failed to read kotadm-redact key in configmap"
		JSON(w, 200, getRedactResponse)
		return
	}

	getRedactResponse.Success = true
	getRedactResponse.UpdatedSpec = encodedData
	JSON(w, 200, getRedactResponse)
}
