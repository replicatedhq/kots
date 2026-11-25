package image

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	imagedocker "go.podman.io/image/v5/docker"
	containerstypes "go.podman.io/image/v5/types"
)

// TestProbeNewImageSource is a temporary diagnostic for the hermetic
// Test_IsPrivateImages rewrite: it records which mock-registry shapes make
// NewImageSource fail (private) or succeed (public). Delete after use.
func TestProbeNewImageSource(t *testing.T) {
	t.Setenv("KOTSADM_INSECURE_SRCREGISTRY", "true")

	sysCtx := &containerstypes.SystemContext{
		DockerDisableV1Ping:         true,
		DockerInsecureSkipTLSVerify: containerstypes.OptionalBoolTrue,
	}

	manifestBody := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":1},"layers":[]}`)
	contentType := "application/vnd.docker.distribution.manifest.v2+json"

	// open registry with one manifest served
	openSrv := newMockManifestRegistry(map[string]mockManifest{
		"library/nginx:1": {body: manifestBody, contentType: contentType},
	})
	defer openSrv.Close()
	openHost := hostFromServer(t, openSrv)

	// every request answered 401, no auth challenge
	bareSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer bareSrv.Close()
	bareHost := hostFromServer(t, bareSrv)

	// every request answered 401 with a bearer challenge pointing back at itself
	var bearerHost string
	bearerSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=\"https://%s/v2/token\",service=\"probe\"", bearerHost))
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer bearerSrv.Close()
	bearerHost = hostFromServer(t, bearerSrv)

	// open ping, but manifests answered 401
	manifestAuthSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" || r.URL.Path == "/v2" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer manifestAuthSrv.Close()
	manifestAuthHost := hostFromServer(t, manifestAuthSrv)

	// closed server -> connection refused
	closedSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedHost := hostFromServer(t, closedSrv)
	closedSrv.Close()

	probe := func(name, image string) {
		imageRef, err := imagedocker.ParseReference(fmt.Sprintf("//%s", image))
		if err != nil {
			t.Logf("%-18s PARSE-ERR %v", name, err)
			return
		}
		src, err := imageRef.NewImageSource(context.Background(), sysCtx)
		if err != nil {
			t.Logf("%-18s SRC-ERR   %v", name, err)
			return
		}
		src.Close()
		t.Logf("%-18s SRC-OK (public)", name)
	}

	probe("open+manifest", fmt.Sprintf("%s/library/nginx:1", openHost))
	probe("open+missing404", fmt.Sprintf("%s/library/nope:1", openHost))
	probe("bare401", fmt.Sprintf("%s/library/nginx:1", bareHost))
	probe("bearer401", fmt.Sprintf("%s/library/nginx:1", bearerHost))
	probe("open+manifest401", fmt.Sprintf("%s/library/nginx:1", manifestAuthHost))
	probe("closed", fmt.Sprintf("%s/library/nginx:1", closedHost))
}
