package handlers

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	v1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	appYamlKey            = "application.yaml"
	iconURI               = "https://cdn2.iconfinder.com/data/icons/mixd/512/16_kubernetes-512.png"
	metadataConfigMapName = "kotsadm-application-metadata"
	upstreamUriKey        = "upstreamUri"
	defaultAppName        = "the application"
)

// MetadataResponse non sensitive information to be used by ui pre-login
type MetadataResponse struct {
	IconURI     string                   `json:"iconUri"`
	Branding    MetadataResponseBranding `json:"branding"`
	Name        string                   `json:"name"`
	Namespace   string                   `json:"namespace"`
	UpstreamURI string                   `json:"upstreamUri"`
	// ConsoleFeatureFlags optional flags from application.yaml used to enable ui features
	ConsoleFeatureFlags  []string             `json:"consoleFeatureFlags"`
	AdminConsoleMetadata AdminConsoleMetadata `json:"adminConsoleMetadata"`
}

type MetadataResponseBranding struct {
	Css       string   `json:"css"`
	FontFaces []string `json:"fontFaces"`
}

type AdminConsoleMetadata struct {
	IsAirgap bool `json:"isAirgap"`
	IsKurl   bool `json:"isKurl"`
}

// GetMetadataHandler helper function that returns a http handler func that returns metadata. It takes a function that
// retrieves state information from an active k8s cluster.
func GetMetadataHandler(getK8sInfoFn MetadataK8sFn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metadataResponse := MetadataResponse{
			IconURI:   iconURI,
			Name:      defaultAppName,
			Namespace: util.PodNamespace,
		}

		brandingConfigMap, kotsadmMetadata, err := getK8sInfoFn()
		if err != nil {
			// if we can't find config map in cluster, it's not an error,  we still want to return a stripped down response
			if kuberneteserrors.IsNotFound(err) {
				metadataResponse.AdminConsoleMetadata.IsAirgap = kotsadmMetadata.IsAirgap
				metadataResponse.AdminConsoleMetadata.IsKurl = kotsadmMetadata.IsKurl

				logger.Info(fmt.Sprintf("config map %q not found", metadataConfigMapName))
				JSON(w, http.StatusOK, &metadataResponse)
				return
			}

			logger.Error(err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		data, ok := brandingConfigMap.Data[appYamlKey]
		if !ok {
			logger.Error(fmt.Errorf("%s key not found in the configmap %s", appYamlKey, metadataConfigMapName))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// parse data as a kotskind
		obj, gvk, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(data), nil, nil)
		if err != nil {
			logger.Error(fmt.Errorf("failed to decode application yaml %w", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Application" {
			logger.Error(fmt.Errorf("expected Application crd but get %#v", gvk))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		application := obj.(*kotsv1beta1.Application)
		metadataResponse.IconURI = application.Spec.Icon
		metadataResponse.Branding = formatBrandingResponse(application.Spec.Branding)
		metadataResponse.Name = application.Spec.Title
		metadataResponse.UpstreamURI = brandingConfigMap.Data[upstreamUriKey]
		metadataResponse.ConsoleFeatureFlags = application.Spec.ConsoleFeatureFlags
		metadataResponse.AdminConsoleMetadata = AdminConsoleMetadata{
			IsAirgap: kotsadmMetadata.IsAirgap,
			IsKurl:   kotsadmMetadata.IsKurl,
		}

		JSON(w, http.StatusOK, metadataResponse)
	}
}

// Converts the application spec branding field into the response format expected by the UI
func formatBrandingResponse(branding kotsv1beta1.ApplicationBranding) MetadataResponseBranding {
	response := MetadataResponseBranding{
		Css:       template.HTMLEscapeString(branding.Css),
		FontFaces: []string{},
	}
	for _, fontFile := range branding.FontFiles {
		if len(fontFile.Sources) == 0 {
			continue
		}
		sources := []string{}
		for _, source := range fontFile.Sources {
			sources = append(sources, fmt.Sprintf(`url("data:font/%s; base64, %s") format("%s")`, source.Format, source.Data, source.Format))
		}
		fontFace := fmt.Sprintf(`@font-face { font-family: "%s"; src: %s; }`, fontFile.FontFamily, strings.Join(sources, ", "))
		response.FontFaces = append(response.FontFaces, fontFace)
	}
	return response
}

// GetMetaDataConfig retrieves configMap from k8s used to construct metadata
func GetMetaDataConfig() (*v1.ConfigMap, types.Metadata, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, types.Metadata{}, nil
	}

	kotsadmMetadata := kotsadm.GetMetadata()

	brandingConfigMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), metadataConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, kotsadmMetadata, err
	}

	return brandingConfigMap, kotsadmMetadata, nil
}

type MetadataK8sFn func() (*v1.ConfigMap, types.Metadata, error)
