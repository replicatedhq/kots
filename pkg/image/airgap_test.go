package image

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/pkg/errors"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/image/types"
	"github.com/stretchr/testify/require"
)

func TestPushEmbeddedClusterArtifacts(t *testing.T) {
	tests := []struct {
		name                 string
		embeddedClusterFiles map[string][]byte
		wantArtifacts        map[string]bool
		wantErr              bool
	}{
		{
			name:                 "no embedded cluster files",
			embeddedClusterFiles: map[string][]byte{},
			wantArtifacts:        map[string]bool{},
			wantErr:              false,
		},
		{
			name: "has embedded cluster files",
			embeddedClusterFiles: map[string][]byte{
				"test-app":         []byte("this-is-the-binary"),
				"charts.tar.gz":    []byte("this-is-the-charts-bundle"),
				"images-amd64.tar": []byte("this-is-the-images-bundle"),
			},
			wantArtifacts: map[string]bool{
				"binary": true,
				"charts": true,
				"images": true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			tmp := t.TempDir()
			airgapBundle := fmt.Sprintf("%s/airgap-bundle", tmp)
			if err := createTestAirgapBundle(airgapBundle, tt.embeddedClusterFiles); err != nil {
				t.Fatalf("Failed to create airgap bundle: %v", err)
			}

			pushedArtifacts := make(map[string]bool)
			mockRegistryServer := newMockRegistryServer(pushedArtifacts)
			defer mockRegistryServer.Close()

			u, err := url.Parse(mockRegistryServer.URL)
			if err != nil {
				t.Fatalf("Failed to parse mock server URL: %v", err)
			}

			opts := types.PushEmbeddedArtifactsOptions{
				Registry: dockerregistrytypes.RegistryOptions{
					Endpoint:  u.Host,
					Namespace: "test-app",
				},
				Tag:        "v1",
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
			req.Equal(pushedArtifacts, tt.wantArtifacts)
		})
	}
}

func createTestAirgapBundle(airgapBundle string, embeddedClusterFiles map[string][]byte) error {
	if err := os.MkdirAll(airgapBundle, 0755); err != nil {
		return errors.Wrap(err, "failed to create airgap bundle directory")
	}

	if len(embeddedClusterFiles) == 0 {
		return nil
	}

	embeddedClusterDir := filepath.Join(airgapBundle, "embedded-cluster")
	if err := os.MkdirAll(embeddedClusterDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create embedded-cluster directory")
	}

	for name, data := range embeddedClusterFiles {
		if err := os.WriteFile(filepath.Join(embeddedClusterDir, name), data, 0644); err != nil {
			return errors.Wrap(err, "failed to write file")
		}
	}

	return nil
}

func newMockRegistryServer(pushedArtifacts map[string]bool) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		blobsRegex := regexp.MustCompile(`/v2/test-app/embedded-cluster/([^/]+)/blobs/(.*)`)
		manifestsRegex := regexp.MustCompile(`/v2/test-app/embedded-cluster/([^/]+)/manifests/(.*)`)

		switch {
		case r.Method == http.MethodHead && blobsRegex.MatchString(r.URL.Path):
			w.Header().Set("Content-Length", "0")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && manifestsRegex.MatchString(r.URL.Path):
			artifact := manifestsRegex.FindStringSubmatch(r.URL.Path)[1] // binary, charts, or images
			pushedArtifacts[artifact] = true
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
