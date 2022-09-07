package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

type mockNotFound struct{}

func (mockNotFound) Error() string         { return "not found" }
func (mockNotFound) Status() metav1.Status { return metav1.Status{Reason: metav1.StatusReasonNotFound} }

func Test_MetadataHandler(t *testing.T) {
	configMap := `apiVersion: v1
data:
  application.yaml: |
    apiVersion: kots.io/v1beta1
    kind: Application
    metadata:
      name: app-slug
    spec:
      icon: https://foo.com/icon.png
      title: App Name
      consoleFeatureFlags: 
         - feature1
         - feature2
    status: {}
kind: ConfigMap
metadata:
  labels:
    app.kubernetes.io/name: kotsadm
    manager: kotsadm
  name: kotsadm-application-metadata
  namespace: default
`

	tests := []struct {
		name       string
		funcPtr    MetadataK8sFn
		httpStatus int
		expected   MetadataResponse
	}{
		{
			name: "happy path feature flag test",
			funcPtr: func() (*v1.ConfigMap, types.Metadata, error) {

				// parse data as a kotskind
				obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(configMap), nil, nil)
				require.Nil(t, err)

				meta := types.Metadata{
					IsKurl: true,
				}
				return obj.(*v1.ConfigMap), meta, nil

			},
			expected: MetadataResponse{
				AdminConsoleMetadata: AdminConsoleMetadata{
					IsKurl: true,
				},
				IconURI: "https://foo.com/icon.png",
				Branding: MetadataResponseBranding{
					Css:       "",
					FontFaces: []string{},
				},
				Name:                "App Name",
				ConsoleFeatureFlags: []string{"feature1", "feature2"},
				Namespace:           util.PodNamespace,
			},
			httpStatus: http.StatusOK,
		},
		{
			name: "cluster error",
			funcPtr: func() (*v1.ConfigMap, types.Metadata, error) {
				return nil, types.Metadata{}, errors.New("wah wah wah")
			},
			httpStatus: http.StatusServiceUnavailable,
		},
		{
			name: "cluster present, no kurl",
			funcPtr: func() (*v1.ConfigMap, types.Metadata, error) {
				return nil, types.Metadata{}, &mockNotFound{}
			},
			httpStatus: http.StatusOK,
			expected: MetadataResponse{
				IconURI:   iconURI,
				Name:      defaultAppName,
				Namespace: util.PodNamespace,
			},
		},
		{
			name: "application branding css only",
			funcPtr: func() (*v1.ConfigMap, types.Metadata, error) {

				cm := `apiVersion: v1
data:
  application.yaml: |
    apiVersion: kots.io/v1beta1
    kind: Application
    metadata:
      name: app-slug
    spec:
      title: App Name
      branding:
        css: "body { background-color: red; }"
    status: {}
kind: ConfigMap
metadata:
  name: kotsadm-application-metadata
`

				// parse data as a kotskind
				obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(cm), nil, nil)
				require.Nil(t, err)

				return obj.(*v1.ConfigMap), types.Metadata{}, nil

			},
			expected: MetadataResponse{
				AdminConsoleMetadata: AdminConsoleMetadata{},
				Name:                 "App Name",
				Branding: MetadataResponseBranding{
					Css:       "body { background-color: red; }",
					FontFaces: []string{},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
		},
		{
			name: "application branding font files only",
			funcPtr: func() (*v1.ConfigMap, types.Metadata, error) {

				cm := `apiVersion: v1
data:
  application.yaml: |
    apiVersion: kots.io/v1beta1
    kind: Application
    metadata:
      name: app-slug
    spec:
      title: App Name
      branding:
        fontFiles:
        - fontFamily: "MyFont"
          sources:
          - format: "woff"
            data: "woff-base64-data"
          - format: "woff2"
            data: "woff2-base64-data"
kind: ConfigMap
metadata:
  name: kotsadm-application-metadata
`

				// parse data as a kotskind
				obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(cm), nil, nil)
				require.Nil(t, err)

				return obj.(*v1.ConfigMap), types.Metadata{}, nil

			},
			expected: MetadataResponse{
				AdminConsoleMetadata: AdminConsoleMetadata{},
				Name:                 "App Name",
				Branding: MetadataResponseBranding{
					Css: "",
					FontFaces: []string{
						`@font-face { font-family: "MyFont"; src: url("data:font/woff; base64, woff-base64-data") format("woff"), url("data:font/woff2; base64, woff2-base64-data") format("woff2"); }`,
					},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
		},
		{
			name: "application branding css and font files",
			funcPtr: func() (*v1.ConfigMap, types.Metadata, error) {

				cm := `apiVersion: v1
data:
  application.yaml: |
    apiVersion: kots.io/v1beta1
    kind: Application
    metadata:
      name: app-slug
    spec:
      title: App Name
      branding:
        css: "body { background-color: red; }"
        fontFiles:
        - fontFamily: "MyFont"
          sources:
          - format: "woff"
            data: "woff-base64-data"
          - format: "woff2"
            data: "woff2-base64-data"
kind: ConfigMap
metadata:
  name: kotsadm-application-metadata
`

				// parse data as a kotskind
				obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(cm), nil, nil)
				require.Nil(t, err)

				return obj.(*v1.ConfigMap), types.Metadata{}, nil

			},
			expected: MetadataResponse{
				AdminConsoleMetadata: AdminConsoleMetadata{},
				Name:                 "App Name",
				Branding: MetadataResponseBranding{
					Css: "body { background-color: red; }",
					FontFaces: []string{
						`@font-face { font-family: "MyFont"; src: url("data:font/woff; base64, woff-base64-data") format("woff"), url("data:font/woff2; base64, woff2-base64-data") format("woff2"); }`,
					},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
		},
		{
			name: "application branding font files with empty sources",
			funcPtr: func() (*v1.ConfigMap, types.Metadata, error) {

				cm := `apiVersion: v1
data:
  application.yaml: |
    apiVersion: kots.io/v1beta1
    kind: Application
    metadata:
      name: app-slug
    spec:
      title: App Name
      branding:
        fontFiles:
        - fontFamily: "MyFont"
          sources:
kind: ConfigMap
metadata:
  name: kotsadm-application-metadata
`

				// parse data as a kotskind
				obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(cm), nil, nil)
				require.Nil(t, err)

				return obj.(*v1.ConfigMap), types.Metadata{}, nil

			},
			expected: MetadataResponse{
				AdminConsoleMetadata: AdminConsoleMetadata{},
				Name:                 "App Name",
				Branding: MetadataResponseBranding{
					Css:       "",
					FontFaces: []string{},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
		},
		{
			name: "application branding css and empty font files",
			funcPtr: func() (*v1.ConfigMap, types.Metadata, error) {

				cm := `apiVersion: v1
data:
  application.yaml: |
    apiVersion: kots.io/v1beta1
    kind: Application
    metadata:
      name: app-slug
    spec:
      title: App Name
      branding:
        css: "body { background-color: red; }"
        fontFiles:
kind: ConfigMap
metadata:
  name: kotsadm-application-metadata
`

				// parse data as a kotskind
				obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(cm), nil, nil)
				require.Nil(t, err)

				return obj.(*v1.ConfigMap), types.Metadata{}, nil

			},
			expected: MetadataResponse{
				AdminConsoleMetadata: AdminConsoleMetadata{},
				Name:                 "App Name",
				Branding: MetadataResponseBranding{
					Css:       "body { background-color: red; }",
					FontFaces: []string{},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
		},
		{
			name: "empty application branding",
			funcPtr: func() (*v1.ConfigMap, types.Metadata, error) {

				cm := `apiVersion: v1
data:
  application.yaml: |
    apiVersion: kots.io/v1beta1
    kind: Application
    metadata:
      name: app-slug
    spec:
      title: App Name
      branding:
kind: ConfigMap
metadata:
  name: kotsadm-application-metadata
`

				// parse data as a kotskind
				obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(cm), nil, nil)
				require.Nil(t, err)

				return obj.(*v1.ConfigMap), types.Metadata{}, nil

			},
			expected: MetadataResponse{
				AdminConsoleMetadata: AdminConsoleMetadata{},
				Name:                 "App Name",
				Branding: MetadataResponseBranding{
					Css:       "",
					FontFaces: []string{},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
		},
		{
			name: "invalid application branding",
			funcPtr: func() (*v1.ConfigMap, types.Metadata, error) {

				cm := `apiVersion: v1
data:
  application.yaml: |
    apiVersion: kots.io/v1beta1
    kind: Application
    metadata:
      name: app-slug
    spec:
      title: App Name
      branding: "this is not a valid branding"
kind: ConfigMap
metadata:
  name: kotsadm-application-metadata
`

				// parse data as a kotskind
				obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(cm), nil, nil)
				require.Nil(t, err)

				return obj.(*v1.ConfigMap), types.Metadata{}, nil

			},
			httpStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ts := httptest.NewServer(GetMetadataHandler(test.funcPtr))
			defer ts.Close()

			response, err := http.Get(ts.URL)
			require.Nil(t, err)
			require.Equal(t, test.httpStatus, response.StatusCode)
			if response.StatusCode != http.StatusOK {
				return
			}
			var metadata MetadataResponse
			require.Nil(t, json.NewDecoder(response.Body).Decode(&metadata))
			require.Equal(t, test.expected, metadata)
		})
	}

}
