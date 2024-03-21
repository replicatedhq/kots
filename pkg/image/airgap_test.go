package image

import (
	"archive/tar"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/image/types"
	"github.com/stretchr/testify/require"
)

func TestPushEmbeddedClusterArtifacts(t *testing.T) {
	tests := []struct {
		name          string
		airgapFiles   map[string][]byte
		wantArtifacts map[string]string
		wantErr       bool
	}{
		{
			name: "no embedded cluster files",
			airgapFiles: map[string][]byte{
				"airgap.yaml":      []byte("this-is-the-airgap-metadata"),
				"app.tar.gz":       []byte("this-is-the-app-archive"),
				"images/something": []byte("this-is-an-image"),
			},
			wantArtifacts: map[string]string{},
			wantErr:       false,
		},
		{
			name: "has embedded cluster files",
			airgapFiles: map[string][]byte{
				"airgap.yaml":                       []byte("this-is-the-airgap-metadata"),
				"app.tar.gz":                        []byte("this-is-the-app-archive"),
				"images/something":                  []byte("this-is-an-image"),
				"embedded-cluster/test-app":         []byte("this-is-the-binary"),
				"embedded-cluster/charts.tar.gz":    []byte("this-is-the-charts-bundle"),
				"embedded-cluster/images-amd64.tar": []byte("this-is-the-images-bundle"),
				"embedded-cluster/some-file-TBD":    []byte("this-is-an-arbitrary-file"),
			},
			wantArtifacts: map[string]string{
				"test-app":         "test-tag",
				"charts.tar.gz":    "test-tag",
				"images-amd64.tar": "test-tag",
				"some-file-tbd":    "test-tag",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			airgapBundle := filepath.Join(t.TempDir(), "application.airgap")
			if err := createTestAirgapBundle(tt.airgapFiles, airgapBundle); err != nil {
				t.Fatalf("Failed to create airgap bundle: %v", err)
			}

			pushedArtifacts := make(map[string]string)
			mockRegistryServer := newMockRegistryServer(pushedArtifacts)
			defer mockRegistryServer.Close()

			u, err := url.Parse(mockRegistryServer.URL)
			if err != nil {
				t.Fatalf("Failed to parse mock server URL: %v", err)
			}

			opts := types.PushEmbeddedClusterArtifactsOptions{
				Registry: dockerregistrytypes.RegistryOptions{
					Endpoint:  u.Host,
					Namespace: "test-app",
				},
				Tag:        "test-tag",
				HTTPClient: mockRegistryServer.Client(),
			}
			if err := PushEmbeddedClusterArtifacts(airgapBundle, opts); (err != nil) != tt.wantErr {
				t.Errorf("PushEmbeddedClusterArtifacts() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}

			// validate that each of the expected artifacts were pushed to the registry
			req.Equal(tt.wantArtifacts, pushedArtifacts)
		})
	}
}

func createTestAirgapBundle(airgapFiles map[string][]byte, dstPath string) error {
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	gzipWriter := gzip.NewWriter(dstFile)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for name, data := range airgapFiles {
		header := &tar.Header{
			Name: name,
			Size: int64(len(data)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write(data); err != nil {
			return err
		}
	}

	return nil
}

func newMockRegistryServer(pushedArtifacts map[string]string) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		blobsRegex := regexp.MustCompile(`/v2/test-app/embedded-cluster/([^/]+)/blobs/(.*)`)
		manifestsRegex := regexp.MustCompile(`/v2/test-app/embedded-cluster/([^/]+)/manifests/(.*)`)

		switch {
		case r.Method == http.MethodHead && blobsRegex.MatchString(r.URL.Path):
			w.Header().Set("Content-Length", "123")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && manifestsRegex.MatchString(r.URL.Path):
			repo := manifestsRegex.FindStringSubmatch(r.URL.Path)[1]
			tag := manifestsRegex.FindStringSubmatch(r.URL.Path)[2]
			pushedArtifacts[repo] = tag
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
