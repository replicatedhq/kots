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
	dockertypes "github.com/replicatedhq/kots/pkg/docker/types"
	"github.com/replicatedhq/kots/pkg/image/types"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
)

func TestPushEmbeddedClusterArtifacts(t *testing.T) {
	testAppSlug := "test-app"
	testChannelID := "test-tag"
	testUpdateCursor := "test-cursor"
	testVersionLabel := "test-version"

	tests := []struct {
		name                  string
		airgapFiles           map[string][]byte
		artifactsToPush       *kotsv1beta1.EmbeddedClusterArtifacts
		useTLS                bool
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
			artifactsToPush:       nil,
			wantRegistryArtifacts: map[string]string{},
			wantErr:               false,
		},
		{
			name: "has embedded cluster files",
			airgapFiles: map[string][]byte{
				"airgap.yaml":                            []byte("this-is-the-airgap-metadata"),
				"app.tar.gz":                             []byte("this-is-the-app-archive"),
				"images/something":                       []byte("this-is-an-image"),
				"embedded-cluster/test-app":              []byte("this-is-the-binary"),
				"embedded-cluster/charts.tar.gz":         []byte("this-is-the-charts-bundle"),
				"embedded-cluster/images-amd64.tar":      []byte("this-is-the-images-bundle"),
				"embedded-cluster/version-metadata.json": []byte("this-is-the-metadata"),
				"embedded-cluster/some-file-TBD":         []byte("this-is-an-arbitrary-file"),
			},
			artifactsToPush: &kotsv1beta1.EmbeddedClusterArtifacts{
				BinaryAmd64: "embedded-cluster/test-app",
				ImagesAmd64: "embedded-cluster/images-amd64.tar",
				Charts:      "embedded-cluster/charts.tar.gz",
				Metadata:    "embedded-cluster/version-metadata.json",
			},
			wantRegistryArtifacts: map[string]string{
				fmt.Sprintf("%s/embedded-cluster/test-app", testAppSlug):              fmt.Sprintf("%s-%s-%s", testChannelID, testUpdateCursor, testVersionLabel),
				fmt.Sprintf("%s/embedded-cluster/charts.tar.gz", testAppSlug):         fmt.Sprintf("%s-%s-%s", testChannelID, testUpdateCursor, testVersionLabel),
				fmt.Sprintf("%s/embedded-cluster/images-amd64.tar", testAppSlug):      fmt.Sprintf("%s-%s-%s", testChannelID, testUpdateCursor, testVersionLabel),
				fmt.Sprintf("%s/embedded-cluster/version-metadata.json", testAppSlug): fmt.Sprintf("%s-%s-%s", testChannelID, testUpdateCursor, testVersionLabel),
			},
			wantErr: false,
		},
		{
			name: "has embedded cluster files and registry has TLS",
			airgapFiles: map[string][]byte{
				"airgap.yaml":                            []byte("this-is-the-airgap-metadata"),
				"app.tar.gz":                             []byte("this-is-the-app-archive"),
				"images/something":                       []byte("this-is-an-image"),
				"embedded-cluster/test-app":              []byte("this-is-the-binary"),
				"embedded-cluster/charts.tar.gz":         []byte("this-is-the-charts-bundle"),
				"embedded-cluster/images-amd64.tar":      []byte("this-is-the-images-bundle"),
				"embedded-cluster/version-metadata.json": []byte("this-is-the-metadata"),
				"embedded-cluster/some-file-TBD":         []byte("this-is-an-arbitrary-file"),
			},
			artifactsToPush: &kotsv1beta1.EmbeddedClusterArtifacts{
				BinaryAmd64: "embedded-cluster/test-app",
				ImagesAmd64: "embedded-cluster/images-amd64.tar",
				Charts:      "embedded-cluster/charts.tar.gz",
				Metadata:    "embedded-cluster/version-metadata.json",
			},
			useTLS: true,
			wantRegistryArtifacts: map[string]string{
				fmt.Sprintf("%s/embedded-cluster/test-app", testAppSlug):              fmt.Sprintf("%s-%s-%s", testChannelID, testUpdateCursor, testVersionLabel),
				fmt.Sprintf("%s/embedded-cluster/charts.tar.gz", testAppSlug):         fmt.Sprintf("%s-%s-%s", testChannelID, testUpdateCursor, testVersionLabel),
				fmt.Sprintf("%s/embedded-cluster/images-amd64.tar", testAppSlug):      fmt.Sprintf("%s-%s-%s", testChannelID, testUpdateCursor, testVersionLabel),
				fmt.Sprintf("%s/embedded-cluster/version-metadata.json", testAppSlug): fmt.Sprintf("%s-%s-%s", testChannelID, testUpdateCursor, testVersionLabel),
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
			mockRegistryServer := newMockRegistryServer(pushedRegistryArtifacts, tt.useTLS)
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
				ChannelID:    testChannelID,
				UpdateCursor: testUpdateCursor,
				VersionLabel: testVersionLabel,
				HTTPClient:   mockRegistryServer.Client(),
			}
			err = PushEmbeddedClusterArtifacts(airgapBundle, tt.artifactsToPush, opts)
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}

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

func newMockRegistryServer(pushedRegistryArtifacts map[string]string, useTLS bool) *httptest.Server {
	newServerFn := httptest.NewServer
	if useTLS {
		newServerFn = httptest.NewTLSServer
	}

	return newServerFn(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func Test_getImageInfosFromBundle(t *testing.T) {
	tests := []struct {
		name        string
		airgapFiles map[string][]byte
		want        map[string]*imagetypes.ImageInfo
		wantErr     bool
	}{
		{
			name: "no images",
			airgapFiles: map[string][]byte{
				"airgap.yaml": []byte("this-is-the-airgap-metadata"),
				"app.tar.gz":  []byte("this-is-the-app-archive"),
			},
			want: map[string]*imagetypes.ImageInfo{},
		},
		{
			name: "has images",
			airgapFiles: map[string][]byte{
				"airgap.yaml":                          []byte("this-is-the-airgap-metadata"),
				"app.tar.gz":                           []byte("this-is-the-app-archive"),
				"images/docker-archive/something":      []byte("this-is-an-image"),
				"images/docker-archive/something-else": []byte("this-is-another-image"),
			},
			want: map[string]*imagetypes.ImageInfo{
				"images/docker-archive/something": &imagetypes.ImageInfo{
					Format: dockertypes.FormatDockerArchive,
					Layers: map[string]*imagetypes.LayerInfo{},
					Status: "queued",
				},
				"images/docker-archive/something-else": &imagetypes.ImageInfo{
					Format: dockertypes.FormatDockerArchive,
					Layers: map[string]*imagetypes.LayerInfo{},
					Status: "queued",
				},
			},
		},
		{
			name: "has images and embedded cluster artifacts",
			airgapFiles: map[string][]byte{
				"airgap.yaml":                          []byte("this-is-the-airgap-metadata"),
				"app.tar.gz":                           []byte("this-is-the-app-archive"),
				"embedded-cluster/test-app":            []byte("this-is-the-binary"),
				"embedded-cluster/charts.tar.gz":       []byte("this-is-the-charts-bundle"),
				"embedded-cluster/images-amd64.tar":    []byte("this-is-the-images-bundle"),
				"images/docker-archive/something":      []byte("this-is-an-image"),
				"images/docker-archive/something-else": []byte("this-is-another-image"),
			},
			want: map[string]*imagetypes.ImageInfo{
				"images/docker-archive/something": &imagetypes.ImageInfo{
					Format: dockertypes.FormatDockerArchive,
					Layers: map[string]*imagetypes.LayerInfo{},
					Status: "queued",
				},
				"images/docker-archive/something-else": &imagetypes.ImageInfo{
					Format: dockertypes.FormatDockerArchive,
					Layers: map[string]*imagetypes.LayerInfo{},
					Status: "queued",
				},
			},
		},
		{
			name: "invalid image path",
			airgapFiles: map[string][]byte{
				"airgap.yaml":                      []byte("this-is-the-airgap-metadata"),
				"app.tar.gz":                       []byte("this-is-the-app-archive"),
				"images/not-within-docker-archive": []byte("this-is-an-image"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			airgapBundle := filepath.Join(t.TempDir(), "application.airgap")
			err := createTestAirgapBundle(tt.airgapFiles, airgapBundle)
			req.NoError(err)

			got, err := getImageInfosFromBundle(airgapBundle, false)
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}
			req.Equal(tt.want, got)
		})
	}
}
