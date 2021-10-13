package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func Test_MetadataHandler(t *testing.T) {

	configMap :=
		`
	apiVersion: v1
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
		expectErr            bool
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
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			req := require.New(t)
			returnedFeatureFlags, err := testServer()

			req.NoError(err)
			req.Equal(test.expectedFeatureFlags, returnedFeatureFlags)

		})
	}

}

func testServer() ([]string, error) {
	ts := httptest.NewServer(GetMetadataHandler(&Handler{}, GetMetaDataConfig))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		log.Fatal(err)
	}
	greeting, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s", greeting)
	return nil, nil
}
