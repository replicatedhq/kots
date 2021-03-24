package ocistore

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	supportbundletypes "github.com/replicatedhq/kots/pkg/supportbundle/types"
	troubleshootredact "github.com/replicatedhq/troubleshoot/pkg/redact"
	"go.uber.org/zap"
)

func (s *OCIStore) DeletePendingSupportBundle(id string) error {
	return ErrNotImplemented
}

func (s *OCIStore) ListSupportBundles(appID string) ([]*supportbundletypes.SupportBundle, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) ListPendingSupportBundlesForApp(appID string) ([]*supportbundletypes.PendingSupportBundle, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) GetSupportBundle(id string) (*supportbundletypes.SupportBundle, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) CreatePendingSupportBundle(id string, appID string, clusterID string) error {
	return ErrNotImplemented
}

func (s *OCIStore) CreateSupportBundle(id string, appID string, archivePath string, marshalledTree []byte) (*supportbundletypes.SupportBundle, error) {

	fileContents, err := ioutil.ReadFile(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive file")
	}

	baseURI := os.Getenv("STORAGE_BASEURI")
	baseURI = strings.TrimSuffix(baseURI, "/")

	// docker images don't allow a large charset
	// so this names it registry.host/base/supportbundle:{bundle-id}
	ref := fmt.Sprintf("%s/supportbundle:%s", strings.TrimPrefix(baseURI, "docker://"), strings.ToLower(id))

	logger.Debug("pushing support bundle to docker registry",
		zap.String("ref", ref))

	options := docker.ResolverOptions{}

	registryHosts := func(host string) ([]docker.RegistryHost, error) {
		registryHost := docker.RegistryHost{
			Client:       http.DefaultClient,
			Host:         host,
			Scheme:       "https",
			Path:         "/v2",
			Capabilities: docker.HostCapabilityPush,
		}

		if os.Getenv("STORAGE_BASEURI_PLAINHTTP") == "true" {
			registryHost.Scheme = "http"
		}

		return []docker.RegistryHost{
			registryHost,
		}, nil
	}

	options.Hosts = registryHosts

	resolver := docker.NewResolver(options)

	memoryStore := content.NewMemoryStore()
	desc := memoryStore.Add("supportbundle.tar.gz", "application/gzip", fileContents)
	pushContents := []ocispec.Descriptor{desc}
	pushedDescriptor, err := oras.Push(context.Background(), resolver, ref, memoryStore, pushContents)
	if err != nil {
		return nil, errors.Wrap(err, "failed to push archive to docker registry")
	}

	logger.Info("pushed support bundle to docker registry",
		zap.String("bundleID", id),
		zap.String("ref", ref),
		zap.String("digest", pushedDescriptor.Digest.String()))

	// return &types.SupportBundle{
	// 	ID: id,
	// }, nil

	return nil, ErrNotImplemented
}

// GetSupportBundle will fetch the bundle archive and return a path to where it
// is stored. The caller is responsible for deleting.
func (s *OCIStore) GetSupportBundleArchive(bundleID string) (string, error) {
	logger.Debug("getting support bundle",
		zap.String("bundleID", bundleID))

	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	fileStore := content.NewFileStore(tmpDir)
	defer fileStore.Close()

	allowedMediaTypes := []string{"application/gzip"}

	options := docker.ResolverOptions{}

	registryHosts := func(host string) ([]docker.RegistryHost, error) {
		registryHost := docker.RegistryHost{
			Client:       http.DefaultClient,
			Host:         host,
			Scheme:       "https",
			Path:         "/v2",
			Capabilities: docker.HostCapabilityResolve | docker.HostCapabilityPull,
		}

		if os.Getenv("STORAGE_BASEURI_PLAINHTTP") == "true" {
			registryHost.Scheme = "http"
		}

		return []docker.RegistryHost{
			registryHost,
		}, nil
	}

	options.Hosts = registryHosts

	resolver := docker.NewResolver(options)

	baseURI := strings.TrimSuffix(os.Getenv("STORAGE_BASEURI"), "/")
	// docker images don't allow a large charset
	// so this names it registry.host/base/supportbundle:{bundle-id}
	ref := fmt.Sprintf("%s/supportbundle:%s", strings.TrimPrefix(baseURI, "docker://"), strings.ToLower(bundleID))

	pulledDescriptor, _, err := oras.Pull(context.Background(), resolver, ref, fileStore, oras.WithAllowedMediaTypes(allowedMediaTypes))
	if err != nil {
		return "", errors.Wrap(err, "failed to pull from registry storage")
	}

	logger.Debug("pulled support bundle from docker registry",
		zap.String("bundleID", bundleID),
		zap.String("ref", ref),
		zap.String("digest", pulledDescriptor.Digest.String()))

	return filepath.Join(tmpDir, "supportbundle.tar.gz"), nil
}

func (s *OCIStore) GetSupportBundleAnalysis(id string) (*supportbundletypes.SupportBundleAnalysis, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) SetSupportBundleAnalysis(id string, insights []byte) error {
	return ErrNotImplemented
}

func (s *OCIStore) GetRedactions(bundleID string) (troubleshootredact.RedactionList, error) {
	return troubleshootredact.RedactionList{}, ErrNotImplemented
}

func (s *OCIStore) SetRedactions(bundleID string, redacts troubleshootredact.RedactionList) error {
	return ErrNotImplemented
}

func (s *OCIStore) GetSupportBundleSpecForApp(id string) (string, error) {
	return "", ErrNotImplemented
}
