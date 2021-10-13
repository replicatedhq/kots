package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func Test_MetadataHandler(t *testing.T) {

	configMap := `apiVersion: v1
data:
  application.yaml: |
    apiVersion: kots.io/v1beta1
    kind: Application
    metadata:
      name: app-slug
    spec:
      icon: https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png
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
		name                 string
		funcPtr              MetadataK8sFn
		httpStatus           int
		expectedFeatureFlags []string
	}{
		{
			name: "happy path feature flag test",
			funcPtr: func() (*v1.ConfigMap, bool, error) {

				// parse data as a kotskind
				obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(configMap), nil, nil)
				require.Nil(t, err)

				return obj.(*v1.ConfigMap), true, nil

			},
			expectedFeatureFlags: []string{"feature1", "feature2"},
			httpStatus: http.StatusOK,
		},
		{
			name: "k8s sad clown",
			funcPtr: func() (*v1.ConfigMap, bool, error) {
				return nil, false, errors.New("wah wah wah")
			},
			httpStatus: http.StatusServiceUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ts := httptest.NewServer(GetMetadataHandler(&Handler{}, test.funcPtr))
			defer ts.Close()

			response, err := http.Get(ts.URL)
			require.Nil(t, err)
			require.Equal(t, test.httpStatus, response.StatusCode)
			if response.StatusCode != http.StatusOK {
				return
			}
			var metadata MetadataResponse
			require.Nil(t, json.NewDecoder(response.Body).Decode(&metadata))
			require.Equal(t, test.expectedFeatureFlags, metadata.ConsoleFeatureFlags)
		})
	}

}
