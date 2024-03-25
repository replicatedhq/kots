package image

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
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
	testAppSlug := "test-app"
	testTag := "test-tag"

	tests := []struct {
		name                  string
		airgapFiles           map[string][]byte
		wantRegistryArtifacts map[string]string
		wantErr               bool
	}{
		{
			name: "no embedded cluster files",
			airgapFiles: map[string][]byte{
				"airgap.yaml":      []byte("this-is-the-airgap-metadata"),
				"app.tar.gz":       []byte("this-is-the-app-archive"),
				"images/something": []byte("this-is-an-image"),
			},
			wantRegistryArtifacts: map[string]string{},
			wantErr:               false,
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
			wantRegistryArtifacts: map[string]string{
				fmt.Sprintf("%s/embedded-cluster/test-app", testAppSlug):         testTag,
				fmt.Sprintf("%s/embedded-cluster/charts.tar.gz", testAppSlug):    testTag,
				fmt.Sprintf("%s/embedded-cluster/images-amd64.tar", testAppSlug): testTag,
				fmt.Sprintf("%s/embedded-cluster/some-file-tbd", testAppSlug):    testTag,
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

			pushedRegistryArtifacts := make(map[string]string)
			mockRegistryServer := newMockRegistryServer(pushedRegistryArtifacts)
			defer mockRegistryServer.Close()

			u, err := url.Parse(mockRegistryServer.URL)
			if err != nil {
				t.Fatalf("Failed to parse mock server URL: %v", err)
			}

			opts := types.PushEmbeddedClusterArtifactsOptions{
				Registry: dockerregistrytypes.RegistryOptions{
					Endpoint:  u.Host,
					Namespace: testAppSlug,
				},
				Tag:        testTag,
				HTTPClient: mockRegistryServer.Client(),
			}
			gotArtifacts, err := PushEmbeddedClusterArtifacts(airgapBundle, opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("PushEmbeddedClusterArtifacts() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}

			wantArtifacts := make([]string, 0)
			for repo, tag := range tt.wantRegistryArtifacts {
				wantArtifacts = append(wantArtifacts, fmt.Sprintf("%s/%s:%s", u.Host, repo, tag))
			}
			req.ElementsMatch(wantArtifacts, gotArtifacts)

			// validate that each of the expected artifacts were pushed to the registry
			req.Equal(tt.wantRegistryArtifacts, pushedRegistryArtifacts)
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

func newMockRegistryServer(pushedRegistryArtifacts map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		blobsRegex := regexp.MustCompile(`/v2/(.+)/blobs/(.*)`)
		manifestsRegex := regexp.MustCompile(`/v2/(.+)/manifests/(.*)`)

		switch {
		case r.Method == http.MethodHead && blobsRegex.MatchString(r.URL.Path):
			w.Header().Set("Content-Length", "123")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && manifestsRegex.MatchString(r.URL.Path):
			repo := manifestsRegex.FindStringSubmatch(r.URL.Path)[1]
			tag := manifestsRegex.FindStringSubmatch(r.URL.Path)[2]
			pushedRegistryArtifacts[repo] = tag
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
