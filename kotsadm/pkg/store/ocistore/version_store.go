package ocistore

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	"github.com/mholt/archiver"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	versiontypes "github.com/replicatedhq/kots/kotsadm/pkg/version/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"go.uber.org/zap"
)

const (
	AppVersionConfigmapPrefix = "kotsadm-appversion-"
)

func (s OCIStore) appVersionConfigMapNameForApp(appID string) (string, error) {
	a, err := s.GetApp(appID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get app")
	}

	return fmt.Sprintf("%s%s", AppVersionConfigmapPrefix, a.Slug), nil
}

func (s OCIStore) getLatestAppVersion(appID string) (*versiontypes.AppVersion, error) {
	configMapName, err := s.appVersionConfigMapNameForApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get appversion config map name")
	}

	configMap, err := s.getConfigmap(configMapName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app version config map")
	}

	if configMap.Data == nil {
		configMap.Data = map[string]string{}
	}

	maxSequence := int64(-1)
	for k := range configMap.Data {
		possibleMaxSequence, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse sequence")
		}
		if possibleMaxSequence > maxSequence {
			maxSequence = possibleMaxSequence
		}
	}

	if maxSequence == int64(-1) {
		return nil, ErrNotFound
	}

	appVersion := versiontypes.AppVersion{}
	if err := json.Unmarshal([]byte(configMap.Data[strconv.FormatInt(maxSequence, 10)]), &appVersion); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal app version")
	}

	return &appVersion, nil
}

func (s OCIStore) IsGitOpsSupportedForVersion(appID string, sequence int64) (bool, error) {
	configMapName, err := s.appVersionConfigMapNameForApp(appID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get appversion config map name")
	}

	configMap, err := s.getConfigmap(configMapName)
	if err != nil {
		return false, errors.Wrap(err, "failed to get app version config map")
	}

	if configMap.Data == nil {
		configMap.Data = map[string]string{}
	}

	sequenceData, ok := configMap.Data[strconv.FormatInt(sequence, 10)]
	if !ok {
		return false, nil // copied from s3pg store, this isn't an error?
	}

	appVersion := versiontypes.AppVersion{}
	if err := json.Unmarshal([]byte(sequenceData), &appVersion); err != nil {
		return false, errors.Wrap(err, "failed to unmarshal app version data")
	}

	return appVersion.KOTSKinds.License.Spec.IsGitOpsSupported, nil
}

func (s OCIStore) IsRollbackSupportedForVersion(appID string, sequence int64) (bool, error) {
	configMapName, err := s.appVersionConfigMapNameForApp(appID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get appversion config map name")
	}

	configMap, err := s.getConfigmap(configMapName)
	if err != nil {
		return false, errors.Wrap(err, "failed to get app version config map")
	}

	if configMap.Data == nil {
		configMap.Data = map[string]string{}
	}

	sequenceData, ok := configMap.Data[strconv.FormatInt(sequence, 10)]
	if !ok {
		return false, nil // copied from s3pg store, this isn't an error?
	}

	appVersion := versiontypes.AppVersion{}
	if err := json.Unmarshal([]byte(sequenceData), &appVersion); err != nil {
		return false, errors.Wrap(err, "failed to unmarshal app version data")
	}

	return appVersion.KOTSKinds.KotsApplication.Spec.AllowRollback, nil
}

func (s OCIStore) IsSnapshotsSupportedForVersion(a *apptypes.App, sequence int64) (bool, error) {
	return false, ErrNotImplemented
}

// CreateAppVersion takes an unarchived app, makes an archive and then uploads it
// to s3 with the appID and sequence specified
func (s OCIStore) CreateAppVersionArchive(appID string, sequence int64, archivePath string) error {
	paths := []string{
		filepath.Join(archivePath, "upstream"),
		filepath.Join(archivePath, "base"),
		filepath.Join(archivePath, "overlays"),
	}

	skippedFilesPath := filepath.Join(archivePath, "skippedFiles")
	if _, err := os.Stat(skippedFilesPath); err == nil {
		paths = append(paths, skippedFilesPath)
	}

	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	defer os.RemoveAll(tmpDir)
	fileToUpload := filepath.Join(tmpDir, "archive.tar.gz")

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Archive(paths, fileToUpload); err != nil {
		return errors.Wrap(err, "failed to create archive")
	}

	storageBaseURI := os.Getenv("STORAGE_BASEURI")
	if storageBaseURI == "" {
		// KOTS 1.15 and earlier only supported s3 and there was no configuration
		storageBaseURI = fmt.Sprintf("s3://%s/%s", os.Getenv("S3_ENDPOINT"), os.Getenv("S3_BUCKET_NAME"))
	}

	ref := refFromAppVersion(appID, sequence, storageBaseURI)

	fileContents, err := ioutil.ReadFile(fileToUpload)
	if err != nil {
		return errors.Wrap(err, "failed to read archive file")
	}

	logger.Debug("pushing app archive to docker registry",
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
	desc := memoryStore.Add(fmt.Sprintf("appversion-%s-%d.tar.gz", appID, sequence), "application/gzip", fileContents)
	pushContents := []ocispec.Descriptor{desc}
	pushedDescriptor, err := oras.Push(context.Background(), resolver, ref, memoryStore, pushContents)
	if err != nil {
		return errors.Wrap(err, "failed to push archive to docker registry")
	}

	logger.Info("pushed app archive to docker registry",
		zap.String("appID", appID),
		zap.Int64("sequence", sequence),
		zap.String("ref", ref),
		zap.String("digest", pushedDescriptor.Digest.String()))

	return nil
}

// GetAppVersionArchive will fetch the archive and return a string that contains a
// directory name where it's extracted into
func (s OCIStore) GetAppVersionArchive(appID string, sequence int64) (string, error) {
	// too noisy
	// logger.Debug("getting app version archive",
	// 	zap.String("appID", appID),
	// 	zap.Int64("sequence", sequence))

	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	storageBaseURI := os.Getenv("STORAGE_BASEURI")
	if storageBaseURI == "" {
		// KOTS 1.15 and earlier only supported s3 and there was no configuration
		storageBaseURI = fmt.Sprintf("s3://%s/%s", os.Getenv("S3_ENDPOINT"), os.Getenv("S3_BUCKET_NAME"))
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
	ref := refFromAppVersion(appID, sequence, storageBaseURI)

	pulledDescriptor, _, err := oras.Pull(context.Background(), resolver, ref, fileStore, oras.WithAllowedMediaTypes(allowedMediaTypes))
	if err != nil {
		return "", errors.Wrap(err, "failed to pull from registry storage")
	}

	logger.Debug("pulled app archive from docker registry",
		zap.String("appID", appID),
		zap.Int64("sequence", sequence),
		zap.String("ref", ref),
		zap.String("digest", pulledDescriptor.Digest.String()))

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(filepath.Join(tmpDir, fmt.Sprintf("appversion-%s-%d.tar.gz", appID, sequence)), tmpDir); err != nil {
		return "", errors.Wrap(err, "failed to unarchive")
	}

	return tmpDir, nil
}

func (s OCIStore) CreateAppVersion(appID string, currentSequence *int64, appName string, appIcon string, kotsKinds *kotsutil.KotsKinds) (int64, error) {
	// NOTE that this experimental store doesn't have a tx and it's possible that this
	// could overwrite if there are multiple updates happening concurrently
	latestAppVersion, err := s.getLatestAppVersion(appID)
	if !s.IsNotFound(err) {
		return int64(0), errors.Wrap(err, "failed to get latest app version")
	}

	newSequence := int64(0)
	if latestAppVersion != nil {
		newSequence = latestAppVersion.Sequence + 1
	}

	appVersion := versiontypes.AppVersion{
		KOTSKinds: kotsKinds,
		CreatedOn: time.Now(),
		Sequence:  newSequence,
	}

	b, err := json.Marshal(appVersion)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal app version")
	}

	configMapName, err := s.appVersionConfigMapNameForApp(appID)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to get app version config map name")
	}

	configMap, err := s.getConfigmap(configMapName)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to get app version config map")
	}

	if configMap.Data == nil {
		configMap.Data = map[string]string{}
	}

	configMap.Data[strconv.FormatInt(newSequence, 10)] = string(b)

	if err := s.updateConfigmap(configMap); err != nil {
		return int64(0), errors.Wrap(err, "failed to update app version configmap")
	}

	return newSequence, nil
}

func (s OCIStore) AddAppVersionToDownstream(appID string, clusterID string, sequence int64, versionLabel string, status string, source string, diffSummary string, diffSummaryError string, commitURL string, gitDeployable bool) error {
	return ErrNotImplemented
}

func (s OCIStore) GetAppVersion(appID string, sequence int64) (*versiontypes.AppVersion, error) {
	configMapName, err := s.appVersionConfigMapNameForApp(appID)
	if err != nil {
		return nil, errors.New("failed to get configmap name for app version")
	}

	configMap, err := s.getConfigmap(configMapName)
	if err != nil {
		return nil, errors.New("failed to get app version config map")
	}

	if configMap.Data == nil {
		return nil, ErrNotFound
	}

	data, ok := configMap.Data[strconv.FormatInt(sequence, 10)]
	if !ok {
		return nil, ErrNotFound
	}

	appVersion := versiontypes.AppVersion{}
	if err := json.Unmarshal([]byte(data), &appVersion); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal app version")
	}

	return &appVersion, nil
}

func (s OCIStore) GetAppVersionsAfter(appID string, sequence int64) ([]*versiontypes.AppVersion, error) {
	return nil, ErrNotImplemented
}

func refFromAppVersion(appID string, sequence int64, baseURI string) string {
	baseURI = strings.TrimSuffix(baseURI, "/")

	// docker images don't allow a large charset
	// so this names it registry.host/base/lower(app-id):sequence
	ref := fmt.Sprintf("%s/%s:%d", strings.TrimPrefix(baseURI, "docker://"), strings.ToLower(appID), sequence)

	return ref
}
