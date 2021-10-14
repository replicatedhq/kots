package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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
			funcPtr: func() (*v1.ConfigMap, bool, error) {

				// parse data as a kotskind
				obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(configMap), nil, nil)
				require.Nil(t, err)

				return obj.(*v1.ConfigMap), true, nil

			},
			expected: MetadataResponse{
				IsKurlEnabled:       true,
				IconURI:             "https://foo.com/icon.png",
				Name:                "App Name",
				ConsoleFeatureFlags: []string{"feature1", "feature2"},
				Namespace:           util.PodNamespace,
			},
			httpStatus: http.StatusOK,
		},
		{
			name: "cluster error",
			funcPtr: func() (*v1.ConfigMap, bool, error) {
				return nil, false, errors.New("wah wah wah")
			},
			httpStatus: http.StatusServiceUnavailable,
		},
		{
			name: "cluster present, no kurl",
			funcPtr: func() (*v1.ConfigMap, bool, error) {
				return nil, false, &mockNotFound{}
			},
			httpStatus: http.StatusOK,
			expected: MetadataResponse{
				IconURI:   iconURI,
				Name:      defaultAppName,
				Namespace: util.PodNamespace,
			},
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
