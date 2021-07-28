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
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	versiontypes "github.com/replicatedhq/kots/pkg/api/version/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	gitopstypes "github.com/replicatedhq/kots/pkg/gitops/types"
	kotsadmconfig "github.com/replicatedhq/kots/pkg/kotsadmconfig"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kustomize"
	"github.com/replicatedhq/kots/pkg/logger"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	"github.com/replicatedhq/kots/pkg/secrets"
	"github.com/replicatedhq/kots/pkg/store/types"
	"go.uber.org/zap"
)

const (
	AppVersionConfigmapPrefix = "kotsadm-appversion-"
)

func (s *OCIStore) appVersionConfigMapNameForApp(appID string) (string, error) {
	a, err := s.GetApp(appID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get app")
	}

	return fmt.Sprintf("%s%s", AppVersionConfigmapPrefix, a.Slug), nil
}

func (s *OCIStore) getLatestAppVersion(appID string) (*versiontypes.AppVersion, error) {
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

func (s *OCIStore) IsIdentityServiceSupportedForVersion(appID string, sequence int64) (bool, error) {
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

	return appVersion.KOTSKinds.Identity != nil, nil
}

func (s *OCIStore) IsRollbackSupportedForVersion(appID string, sequence int64) (bool, error) {
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

func (s *OCIStore) IsSnapshotsSupportedForVersion(a *apptypes.App, sequence int64, renderer rendertypes.Renderer) (bool, error) {
	return false, ErrNotImplemented
}

// CreateAppVersion takes an unarchived app, makes an archive and then uploads it
// to s3 with the appID and sequence specified
func (s *OCIStore) CreateAppVersionArchive(appID string, sequence int64, archivePath string) error {
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
func (s *OCIStore) GetAppVersionArchive(appID string, sequence int64, dstPath string) error {
	// too noisy
	// logger.Debug("getting app version archive",
	// 	zap.String("appID", appID),
	// 	zap.Int64("sequence", sequence))

	storageBaseURI := os.Getenv("STORAGE_BASEURI")
	if storageBaseURI == "" {
		storageBaseURI = fmt.Sprintf("s3://%s/%s", os.Getenv("S3_ENDPOINT"), os.Getenv("S3_BUCKET_NAME"))
	}

	fileStore := content.NewFileStore(dstPath)
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
		return errors.Wrap(err, "failed to pull from registry storage")
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
	if err := tarGz.Unarchive(filepath.Join(dstPath, fmt.Sprintf("appversion-%s-%d.tar.gz", appID, sequence)), dstPath); err != nil {
		return errors.Wrap(err, "failed to unarchive")
	}

	return nil
}

func (s *OCIStore) CreateAppVersion(appID string, currentSequence *int64, filesInDir string, source string, skipPreflights bool, gitops gitopstypes.DownstreamGitOps) (int64, error) {
	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(filesInDir)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to read kots kinds")
	}

	appName := kotsKinds.KotsApplication.Spec.Title

	a, err := s.GetApp(appID)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to get app")
	}

	if appName == "" {
		appName = a.Name
	}

	appIcon := kotsKinds.KotsApplication.Spec.Icon

	if err := secrets.ReplaceSecretsInPath(filesInDir); err != nil {
		return int64(0), errors.Wrap(err, "failed to replace secrets")
	}

	newSequence, err := s.createAppVersion(appID, currentSequence, appName, appIcon, kotsKinds)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to create app version")
	}

	if err := s.CreateAppVersionArchive(appID, int64(newSequence), filesInDir); err != nil {
		return int64(0), errors.Wrap(err, "failed to create app version archive")
	}

	previousArchiveDir := ""
	if currentSequence != nil {
		previousDir, err := ioutil.TempDir("", "kotsadm")
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(previousDir)

		// Get the previous archive, we need this to calculate the diff
		err = s.GetAppVersionArchive(appID, *currentSequence, previousDir)
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to get previous archive")
		}

		previousArchiveDir = previousDir
	}

	registrySettings, err := s.GetRegistryDetailsForApp(appID)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to get app registry info")
	}

	downstreams, err := s.ListDownstreamsForApp(appID)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to list downstreams")
	}

	for _, d := range downstreams {
		// there's a small chance this is not optimal, but no current code path
		// will support multiple downstreams, so this is cleaner here for now

		downstreamStatus := types.VersionPending
		if currentSequence == nil && kotsKinds.IsConfigurable() { // initial version should always require configuration (if exists) even if all required items are already set and have values (except for automated installs, which can override this later)
			downstreamStatus = types.VersionPendingConfig
		} else if kotsKinds.HasPreflights() && !skipPreflights {
			downstreamStatus = types.VersionPendingPreflight
		}
		if currentSequence != nil { // only check if the version needs configuration for later versions (not the initial one) since the config is always required for the initial version (except for automated installs, which can override that later)
			// check if version needs additional configuration
			t, err := kotsadmconfig.NeedsConfiguration(kotsKinds, registrySettings)
			if err != nil {
				return int64(0), errors.Wrap(err, "failed to check if version needs configuration")
			}
			if t {
				downstreamStatus = types.VersionPendingConfig
			}
		}

		diffSummary, diffSummaryError := "", ""
		if currentSequence != nil {
			// diff this release from the last release
			diff, err := kustomize.DiffAppVersionsForDownstream(d.Name, filesInDir, previousArchiveDir, kotsKinds.KustomizeVersion())
			if err != nil {
				diffSummaryError = errors.Wrap(err, "failed to diff").Error()
			} else {
				b, err := json.Marshal(diff)
				if err != nil {
					diffSummaryError = errors.Wrap(err, "failed to marshal diff").Error()
				}
				diffSummary = string(b)
			}
		}

		commitURL, err := gitops.CreateGitOpsDownstreamCommit(appID, d.ClusterID, int(newSequence), filesInDir, d.Name)
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to create gitops commit")
		}

		err = s.addAppVersionToDownstream(appID, d.ClusterID, newSequence,
			kotsKinds.Installation.Spec.VersionLabel, downstreamStatus, source,
			diffSummary, diffSummaryError, commitURL, commitURL != "")
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to create downstream version")
		}

		// update metadata configmap
		applicationSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Application")
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to marshal application spec")
		}

		if err := s.ensureApplicationMetadata(applicationSpec, os.Getenv("POD_NAMESPACE"), a.UpstreamURI); err != nil {
			return int64(0), errors.Wrap(err, "failed to get metadata config map")
		}
	}

	return newSequence, nil
}

func (s *OCIStore) createAppVersion(appID string, currentSequence *int64, appName string, appIcon string, kotsKinds *kotsutil.KotsKinds) (int64, error) {
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

func (s *OCIStore) addAppVersionToDownstream(appID string, clusterID string, sequence int64, versionLabel string, status types.DownstreamVersionStatus, source string, diffSummary string, diffSummaryError string, commitURL string, gitDeployable bool) error {
	return ErrNotImplemented
}

func (s *OCIStore) GetAppVersion(appID string, sequence int64) (*versiontypes.AppVersion, error) {
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
	appVersion.AppID = appID

	return &appVersion, nil
}

func (s *OCIStore) GetAppVersionsAfter(appID string, sequence int64) ([]*versiontypes.AppVersion, error) {
	return nil, ErrNotImplemented
}

func refFromAppVersion(appID string, sequence int64, baseURI string) string {
	baseURI = strings.TrimSuffix(baseURI, "/")

	// docker images don't allow a large charset
	// so this names it registry.host/base/lower(app-id):sequence
	ref := fmt.Sprintf("%s/%s:%d", strings.TrimPrefix(baseURI, "docker://"), strings.ToLower(appID), sequence)

	return ref
}

func (s *OCIStore) UpdateAppVersionInstallationSpec(appID string, sequence int64, installation kotsv1beta1.Installation) error {
	return ErrNotImplemented
}
