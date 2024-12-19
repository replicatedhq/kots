package handlers

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	v1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	appYamlKey                  = "application.yaml"
	iconURI                     = "https://cdn2.iconfinder.com/data/icons/mixd/512/16_kubernetes-512.png"
	metadataConfigMapName       = "kotsadm-application-metadata"
	upstreamUriKey              = "upstreamUri"
	defaultAppName              = "the application"
	ecRestoreStateConfigMapName = "embedded-cluster-restore-state"
)

// MetadataResponse non sensitive information to be used by ui pre-login
type MetadataResponse struct {
	IconURI     string                   `json:"iconUri"`
	Branding    MetadataResponseBranding `json:"branding"`
	Name        string                   `json:"name"`
	Namespace   string                   `json:"namespace"`
	UpstreamURI string                   `json:"upstreamUri"`
	// ConsoleFeatureFlags optional flags from application.yaml used to enable ui features
	ConsoleFeatureFlags              []string             `json:"consoleFeatureFlags"`
	AdminConsoleMetadata             AdminConsoleMetadata `json:"adminConsoleMetadata"`
	IsEmbeddedClusterWaitingForNodes bool                 `json:"isEmbeddedClusterWaitingForNodes"`
}

type MetadataResponseBranding struct {
	Css       []string `json:"css"`
	FontFaces []string `json:"fontFaces"`
}

type AdminConsoleMetadata struct {
	IsAirgap          bool `json:"isAirgap"`
	IsKurl            bool `json:"isKurl"`
	IsEmbeddedCluster bool `json:"isEmbeddedCluster"`
	IsEC2Install      bool `json:"isEC2Install"`
}

// GetMetadataHandler helper function that returns a http handler func that returns metadata. It takes a function that
// retrieves state information from an active k8s cluster.
func GetMetadataHandler(getK8sInfoFn MetadataK8sFn, kotsStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID := r.FormValue("app_id")

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
				metadataResponse.AdminConsoleMetadata.IsEmbeddedCluster = kotsadmMetadata.IsEmbeddedCluster
				metadataResponse.AdminConsoleMetadata.IsEC2Install = util.IsEC2Install()

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
		metadataResponse.Branding = getBrandingResponse(kotsStore, appID)
		metadataResponse.Name = application.Spec.Title
		metadataResponse.UpstreamURI = brandingConfigMap.Data[upstreamUriKey]
		metadataResponse.ConsoleFeatureFlags = application.Spec.ConsoleFeatureFlags
		metadataResponse.AdminConsoleMetadata = AdminConsoleMetadata{
			IsAirgap:          kotsadmMetadata.IsAirgap,
			IsKurl:            kotsadmMetadata.IsKurl,
			IsEmbeddedCluster: kotsadmMetadata.IsEmbeddedCluster,
			IsEC2Install:      util.IsEC2Install(),
		}

		if kotsadmMetadata.IsEmbeddedCluster {
			clientset, err := k8sutil.GetClientset()
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to get k8s clientset"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			metadataResponse.IsEmbeddedClusterWaitingForNodes, err = isEmbeddedClusterWaitingForNodes(r.Context(), clientset)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to check if embedded cluster restore is in progress"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		JSON(w, http.StatusOK, metadataResponse)
	}
}

// Converts the application spec branding field into the response format expected by the UI
func getBrandingResponse(kotsStore store.Store, appID string) MetadataResponseBranding {
	response := MetadataResponseBranding{
		Css:       []string{},
		FontFaces: []string{},
	}

	var brandingArchive []byte
	if appID != "" {
		latestBrandingArchive, err := kotsStore.GetLatestBrandingForApp(appID)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to get latest branding for app %s", appID))
			return response
		}
		brandingArchive = latestBrandingArchive
	} else {
		latestBrandingArchive, err := kotsStore.GetLatestBranding()
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get latest branding"))
			return response
		}
		brandingArchive = latestBrandingArchive
	}

	if len(brandingArchive) == 0 {
		return response
	}

	tmpDir, err := ioutil.TempDir("", "kotsadm-branding")
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create temp dir"))
		return response
	}
	defer os.RemoveAll(tmpDir)

	if err := ioutil.WriteFile(filepath.Join(tmpDir, "branding.tar.gz"), brandingArchive, 0644); err != nil {
		logger.Error(errors.Wrap(err, "failed to write branding archive to temp dir"))
		return response
	}

	if err := util.ExtractTGZArchive(filepath.Join(tmpDir, "branding.tar.gz"), tmpDir); err != nil {
		logger.Error(errors.Wrap(err, "failed to extract branding archive"))
		return response
	}

	applicationYaml, err := os.ReadFile(filepath.Join(tmpDir, "application.yaml"))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to read application.yaml"))
		return response
	}

	application, err := kotsutil.LoadKotsAppFromContents(applicationYaml)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse kots app from application.yaml"))
		return response
	}

	for _, source := range application.Spec.Branding.Css {
		ext := filepath.Ext(source)

		if ext != ".css" {
			logger.Error(fmt.Errorf("expected css file but got %s", source))
			continue
		}

		contents, err := os.ReadFile(filepath.Join(tmpDir, source))
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to read font file %s", source))
			continue
		}

		response.Css = append(response.Css, string(contents))
	}

	for _, font := range application.Spec.Branding.Fonts {
		if len(font.Sources) == 0 {
			continue
		}

		sources := []string{}
		for _, source := range font.Sources {
			ext := filepath.Ext(source)

			format, ok := kotsutil.BrandingFontFileExtensions[ext]
			if !ok {
				logger.Error(fmt.Errorf("invalid branding file extension %s", ext))
				continue
			}

			contents, err := os.ReadFile(filepath.Join(tmpDir, source))
			if err != nil {
				logger.Error(errors.Wrapf(err, "failed to read font file %s", source))
				continue
			}

			sources = append(sources, fmt.Sprintf(`url("data:font/%s;base64,%s") format("%s")`, format, string(contents), format))
		}

		if len(sources) == 0 {
			continue
		}

		fontFace := fmt.Sprintf(`@font-face { font-family: "%s"; src: %s; }`, font.FontFamily, strings.Join(sources, ", "))
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

	kotsadmMetadata := kotsadm.GetMetadata(clientset)

	brandingConfigMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), metadataConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, kotsadmMetadata, err
	}

	return brandingConfigMap, kotsadmMetadata, nil
}

type MetadataK8sFn func() (*v1.ConfigMap, types.Metadata, error)

func isEmbeddedClusterWaitingForNodes(ctx context.Context, clientset kubernetes.Interface) (bool, error) {
	cm, err := clientset.CoreV1().ConfigMaps("embedded-cluster").Get(ctx, ecRestoreStateConfigMapName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to get configmap %s", ecRestoreStateConfigMapName)
	}
	if cm.Data == nil {
		return false, nil
	}
	state, ok := cm.Data["state"]
	if !ok {
		return false, nil
	}
	return state == "wait-for-nodes", nil
}
