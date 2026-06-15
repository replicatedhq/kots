package image

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	godigest "github.com/opencontainers/go-digest"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.podman.io/image/v5/transports/alltransports"
	containerstypes "go.podman.io/image/v5/types"
)

func Test_IsPrivateImages(t *testing.T) {
	type args struct {
		baseImages      []string
		kotsKindsImages []string
		kotsKinds       *kotsutil.KotsKinds
	}

	tests := []struct {
		image string
		want  bool
	}{
		{
			image: "registry.replicated.com/appslug/image:version",
			want:  true,
		},
		{
			image: "quay.io/replicatedcom/qa-kots-2:alpine-3.4",
			want:  true,
		},
		{
			image: "quay.io/replicatedcom/qa-kots-1:alpine-3.5",
			want:  true,
		},
		{
			image: "quay.io/replicatedcom/qa-kots-3:alpine-3.6",
			want:  true,
		},
		{
			image: "quay.io/replicatedcom/someimage:1@sha256:25dedae0aceb6b4fe5837a0acbacc6580453717f126a095aa05a3c6fcea14dd4",
			want:  true,
		},
		{
			image: "testing.registry.com:5000/testing-ns/random-image:2",
			want:  true,
		},
		{
			image: "testing.registry.com:5000/testing-ns/random-image:1",
			want:  true,
		},
		{
			image: "redis:7@sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
			want:  false,
		},
		{
			image: "nginx:1",
			want:  false,
		},
		{
			image: "busybox",
			want:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.image, func(t *testing.T) {
			req := require.New(t)

			got, err := IsPrivateImage(test.image, dockerregistrytypes.RegistryOptions{})
			req.NoError(err)

			assert.Equal(t, test.want, got)
		})
	}
}

// mockManifest holds the raw bytes and MIME type a fake registry returns for a
// /v2/<name>/manifests/<reference> request.
type mockManifest struct {
	body        []byte
	contentType string
}

// newMockManifestRegistry builds an httptest.Server that responds to the small
// subset of the v2 registry protocol exercised by destinationManifestMatches:
//   - GET /v2/                            → 200 (registry version probe)
//   - GET|HEAD /v2/<name>/manifests/<ref> → manifest body, or 404 if absent
//
// Manifests are keyed by `"<name>:<ref>"`. The server uses TLS so the docker
// transport accepts it; tests must set DestSkipTLSVerify/SrcSkipTLSVerify.
func newMockManifestRegistry(manifests map[string]mockManifest) *httptest.Server {
	manifestsRegex := regexp.MustCompile(`^/v2/(.+)/manifests/(.+)$`)

	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" || r.URL.Path == "/v2" {
			w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
			w.WriteHeader(http.StatusOK)
			return
		}

		m := manifestsRegex.FindStringSubmatch(r.URL.Path)
		if m == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		key := fmt.Sprintf("%s:%s", m[1], m[2])
		entry, ok := manifests[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", entry.contentType)
		w.Header().Set("Docker-Content-Digest", godigest.FromBytes(entry.body).String())
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = w.Write(entry.body)
		}
	}))
}

// hostFromServer strips the scheme from an httptest server URL so it can be
// embedded in a docker:// image reference.
func hostFromServer(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	return u.Host
}

func Test_destinationManifestMatches(t *testing.T) {
	// Two distinct manifest bodies — same media type — produce different digests.
	srcBody := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":1},"layers":[]}`)
	otherBody := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":1},"layers":[]}`)
	contentType := "application/vnd.docker.distribution.manifest.v2+json"

	tests := []struct {
		name        string
		manifests   map[string]mockManifest
		destPath    string // "<name>:<tag>" path on the mock
		wantMatches bool
		wantErr     bool
	}{
		{
			name: "destination matches source — skip push",
			manifests: map[string]mockManifest{
				"src/app:v1": {body: srcBody, contentType: contentType},
				"dst/app:v1": {body: srcBody, contentType: contentType},
			},
			destPath:    "dst/app:v1",
			wantMatches: true,
		},
		{
			name: "destination differs from source — proceed with push",
			manifests: map[string]mockManifest{
				"src/app:v1": {body: srcBody, contentType: contentType},
				"dst/app:v1": {body: otherBody, contentType: contentType},
			},
			destPath:    "dst/app:v1",
			wantMatches: false,
		},
		{
			name: "destination tag missing — proceed with push",
			manifests: map[string]mockManifest{
				"src/app:v1": {body: srcBody, contentType: contentType},
				// dst/app:v1 intentionally absent → 404
			},
			destPath:    "dst/app:v1",
			wantMatches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newMockManifestRegistry(tt.manifests)
			defer srv.Close()
			host := hostFromServer(t, srv)

			srcRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s/src/app:v1", host))
			require.NoError(t, err)
			destRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s/%s", host, tt.destPath))
			require.NoError(t, err)

			opts := imagetypes.CopyImageOptions{
				SrcRef:  srcRef,
				DestRef: destRef,
			}
			srcCtx := &containerstypes.SystemContext{
				DockerInsecureSkipTLSVerify: containerstypes.OptionalBoolTrue,
				DockerDisableV1Ping:         true,
			}
			destCtx := &containerstypes.SystemContext{
				DockerInsecureSkipTLSVerify: containerstypes.OptionalBoolTrue,
				DockerDisableV1Ping:         true,
			}

			got, err := destinationManifestMatches(context.Background(), opts, srcCtx, destCtx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantMatches, got)
		})
	}
}

// Test_destinationManifestMatches_SourceUnreachable verifies the precheck returns
// an error when the source registry cannot serve the manifest, since the caller
// cannot proceed with a copy without a readable source either.
func Test_destinationManifestMatches_SourceUnreachable(t *testing.T) {
	body := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","digest":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":1},"layers":[]}`)
	// Mock holds dest only — source ref will point at a closed port to force
	// NewImageSource to fail.
	srv := newMockManifestRegistry(map[string]mockManifest{
		"dst/app:v1": {body: body, contentType: "application/vnd.docker.distribution.manifest.v2+json"},
	})
	defer srv.Close()
	host := hostFromServer(t, srv)

	srcRef, err := alltransports.ParseImageName("docker://127.0.0.1:1/src/app:v1")
	require.NoError(t, err)
	destRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s/dst/app:v1", host))
	require.NoError(t, err)

	srcCtx := &containerstypes.SystemContext{
		DockerInsecureSkipTLSVerify: containerstypes.OptionalBoolTrue,
		DockerDisableV1Ping:         true,
	}
	destCtx := &containerstypes.SystemContext{
		DockerInsecureSkipTLSVerify: containerstypes.OptionalBoolTrue,
		DockerDisableV1Ping:         true,
	}

	_, err = destinationManifestMatches(context.Background(), imagetypes.CopyImageOptions{
		SrcRef:  srcRef,
		DestRef: destRef,
	}, srcCtx, destCtx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source")
}

// Test_destinationManifestMatches_DestUnreachable verifies the precheck returns
// (false, nil) — not an error — when the destination registry is not reachable.
// The caller should always fall back to a normal push in that case.
func Test_destinationManifestMatches_DestUnreachable(t *testing.T) {
	srv := newMockManifestRegistry(map[string]mockManifest{
		"src/app:v1": {
			body:        []byte(`{"schemaVersion":2}`),
			contentType: "application/vnd.docker.distribution.manifest.v2+json",
		},
	})
	defer srv.Close()
	host := hostFromServer(t, srv)

	srcRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s/src/app:v1", host))
	require.NoError(t, err)
	// Point dest at a port that won't accept connections.
	destRef, err := alltransports.ParseImageName("docker://127.0.0.1:1/dst/app:v1")
	require.NoError(t, err)

	srcCtx := &containerstypes.SystemContext{
		DockerInsecureSkipTLSVerify: containerstypes.OptionalBoolTrue,
		DockerDisableV1Ping:         true,
	}
	destCtx := &containerstypes.SystemContext{
		DockerInsecureSkipTLSVerify: containerstypes.OptionalBoolTrue,
		DockerDisableV1Ping:         true,
	}

	got, err := destinationManifestMatches(context.Background(), imagetypes.CopyImageOptions{
		SrcRef:  srcRef,
		DestRef: destRef,
	}, srcCtx, destCtx)
	require.NoError(t, err)
	assert.False(t, got)
}

