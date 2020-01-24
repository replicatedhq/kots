package handlers

import (
	"net/http"
	"os"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type MetadataResponse struct {
	IconURI       string `json:"iconUri"`
	Name          string `json:"name"`
	Namespace     string `json:"namespace"`
	IsKurlEnabled bool   `json:"isKurlEnabled"`
}

func Metadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	// This is not an authenticated request

	cfg, err := config.GetConfig()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	isKurlEnabled := false
	_, err = clientset.CoreV1().ConfigMaps("kube-system").Get("kurl-config", metav1.GetOptions{})
	if err == nil {
		isKurlEnabled = true
	}

	brandingConfigMap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get("kotsadm-application-metadata", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	metadataResponse := MetadataResponse{
		IconURI:       "https://cdn2.iconfinder.com/data/icons/mixd/512/16_kubernetes-512.png",
		Name:          "the application",
		Namespace:     os.Getenv("POD_NAMESPACE"),
		IsKurlEnabled: isKurlEnabled,
	}

	if err == nil {
		data, ok := brandingConfigMap.Data["application.yaml"]
		if !ok {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}

		// parse data as a kotskind
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode([]byte(data), nil, nil)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}

		if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Application" {
			logger.Error(errors.New("unexpected gvk found in metadata"))
			w.WriteHeader(500)
			return
		}

		application := obj.(*kotsv1beta1.Application)
		metadataResponse.IconURI = application.Spec.Icon
		metadataResponse.Name = application.Spec.Title
	}

	JSON(w, 200, metadataResponse)
}
