package handlers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name                 string
		funcPtr              MetadataK8sFn
		httpStatus           int
		expected             MetadataResponse
		getBrandingArchiveFn func() ([]byte, error)
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
					Css:       []string{},
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
        css:
        - "styles/my-branding.css"
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
					Css: []string{
						"body { background-color: red; }",
					},
					FontFaces: []string{},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
			getBrandingArchiveFn: func() ([]byte, error) {
				files := []brandingArchiveFile{
					{
						name: "styles/my-branding.css",
						data: []byte("body { background-color: red; }"),
					},
					{
						name: "application.yaml",
						data: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
    name: app-slug
spec:
    title: App Name
    branding:
        css:
        - "styles/my-branding.css"`),
					},
				}
				b, err := createBrandingArchiveWithFiles(files)
				if err != nil {
					return nil, err
				}

				return b.Bytes(), nil
			},
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
        fonts:
        - fontFamily: "MyFont"
          sources:
          - "fonts/MyFont.woff"
          - "fonts/MyFont.woff2"
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
					Css: []string{},
					FontFaces: []string{
						`@font-face { font-family: "MyFont"; src: url("data:font/woff;base64,woff-base64-data") format("woff"), url("data:font/woff2;base64,woff2-base64-data") format("woff2"); }`,
					},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
			getBrandingArchiveFn: func() ([]byte, error) {
				files := []brandingArchiveFile{
					{
						name: "fonts/MyFont.woff",
						data: []byte("woff-base64-data"),
					},
					{
						name: "fonts/MyFont.woff2",
						data: []byte("woff2-base64-data"),
					},
					{
						name: "application.yaml",
						data: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
    name: app-slug
spec:
    title: App Name
    branding:
        fonts:
        - fontFamily: "MyFont"
          sources:
          - "fonts/MyFont.woff"
          - "fonts/MyFont.woff2"`),
					},
				}
				b, err := createBrandingArchiveWithFiles(files)
				if err != nil {
					return nil, err
				}

				return b.Bytes(), nil
			},
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
        css:
        - "styles/my-branding.css"
        fonts:
        - fontFamily: "MyFont"
          sources:
          - "fonts/MyFont.woff"
          - "fonts/MyFont.woff2"
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
					Css: []string{
						"body { background-color: red; }",
					},
					FontFaces: []string{
						`@font-face { font-family: "MyFont"; src: url("data:font/woff;base64,woff-base64-data") format("woff"), url("data:font/woff2;base64,woff2-base64-data") format("woff2"); }`,
					},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
			getBrandingArchiveFn: func() ([]byte, error) {
				files := []brandingArchiveFile{
					{
						name: "fonts/MyFont.woff",
						data: []byte("woff-base64-data"),
					},
					{
						name: "fonts/MyFont.woff2",
						data: []byte("woff2-base64-data"),
					},
					{
						name: "styles/my-branding.css",
						data: []byte("body { background-color: red; }"),
					},
					{
						name: "application.yaml",
						data: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
    name: app-slug
spec:
    title: App Name
    branding:
        css:
        - "styles/my-branding.css"
        fonts:
        - fontFamily: "MyFont"
          sources:
          - "fonts/MyFont.woff"
          - "fonts/MyFont.woff2"`),
					},
				}
				b, err := createBrandingArchiveWithFiles(files)
				if err != nil {
					return nil, err
				}

				return b.Bytes(), nil
			},
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
        fonts:
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
					Css:       []string{},
					FontFaces: []string{},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
			getBrandingArchiveFn: func() ([]byte, error) {
				files := []brandingArchiveFile{
					{
						name: "fonts/MyFont.woff",
						data: []byte("woff-base64-data"),
					},
					{
						name: "fonts/MyFont.woff2",
						data: []byte("woff2-base64-data"),
					},
					{
						name: "application.yaml",
						data: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
    name: app-slug
spec:
    title: App Name
    branding:
        fonts:
        - fontFamily: "MyFont"
          sources:`),
					},
				}
				b, err := createBrandingArchiveWithFiles(files)
				if err != nil {
					return nil, err
				}

				return b.Bytes(), nil
			},
		},
		{
			name: "application branding css and empty fonts",
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
        css:
        - "styles/my-branding.css"
        fonts:
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
					Css: []string{
						"body { background-color: red; }",
					},
					FontFaces: []string{},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
			getBrandingArchiveFn: func() ([]byte, error) {
				files := []brandingArchiveFile{
					{
						name: "styles/my-branding.css",
						data: []byte("body { background-color: red; }"),
					},
					{
						name: "application.yaml",
						data: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
    name: app-slug
spec:
    title: App Name
    branding:
        css:
        - "styles/my-branding.css"
        fonts:`),
					},
				}
				b, err := createBrandingArchiveWithFiles(files)
				if err != nil {
					return nil, err
				}

				return b.Bytes(), nil
			},
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
					Css:       []string{},
					FontFaces: []string{},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
			getBrandingArchiveFn: func() ([]byte, error) {
				files := []brandingArchiveFile{
					{
						name: "application.yaml",
						data: []byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
    name: app-slug
spec:
    title: App Name
    branding:`),
					},
				}
				b, err := createBrandingArchiveWithFiles(files)
				if err != nil {
					return nil, err
				}

				return b.Bytes(), nil
			},
		},
		{
			name: "nil branding archive",
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
        css:
        - "styles/my-branding.css"
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
					Css:       []string{},
					FontFaces: []string{},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
			getBrandingArchiveFn: func() ([]byte, error) {
				return nil, nil
			},
		},
		{
			name: "error when getting branding archive",
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
        css:
        - "styles/my-branding.css"
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
					Css:       []string{},
					FontFaces: []string{},
				},
				Namespace: util.PodNamespace,
			},
			httpStatus: http.StatusOK,
			getBrandingArchiveFn: func() ([]byte, error) {
				return nil, errors.New("failed to get branding archive")
			},
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
			mockStore := mock_store.NewMockStore(ctrl)

			mockStore.EXPECT().GetLatestBranding().AnyTimes().DoAndReturn(func() ([]byte, error) {
				if test.getBrandingArchiveFn != nil {
					return test.getBrandingArchiveFn()
				}
				return nil, nil
			})

			ts := httptest.NewServer(GetMetadataHandler(test.funcPtr, mockStore))
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

type brandingArchiveFile struct {
	name string
	data []byte
}

func createBrandingArchiveWithFiles(files []brandingArchiveFile) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	gz := gzip.NewWriter(buf)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	for _, file := range files {
		hdr := &tar.Header{
			Name: file.name,
			Mode: 0600,
			Size: int64(len(file.data)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write(file.data); err != nil {
			return nil, err
		}
	}

	return buf, nil
}

func Test_isEmbeddedClusterRestoreInProgress(t *testing.T) {
	tests := []struct {
		name      string
		clientset kubernetes.Interface
		want      bool
		wantErr   bool
	}{
		{
			name:      "no restore in progress",
			clientset: fake.NewSimpleClientset(),
			want:      false,
			wantErr:   false,
		},
		{
			name: "restore in progress",
			clientset: fake.NewSimpleClientset(&v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      embeddedClusterRestoreConfigMapName,
					Namespace: "embedded-cluster",
				},
			}),
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := isEmbeddedClusterRestoreInProgress(ctx, tt.clientset)
			if (err != nil) != tt.wantErr {
				t.Errorf("isEmbeddedClusterRestoreInProgress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isEmbeddedClusterRestoreInProgress() = %v, want %v", got, tt.want)
			}
		})
	}
}
