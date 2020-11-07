package ocistore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	"github.com/mholt/archiver"
	"github.com/ocidb/ocidb/pkg/ocidb"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	kotsconfig "github.com/replicatedhq/kots/kotsadm/pkg/config"
	gitopstypes "github.com/replicatedhq/kots/kotsadm/pkg/gitops/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/render"
	versiontypes "github.com/replicatedhq/kots/kotsadm/pkg/version/types"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kustomize"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func (s OCIStore) IsGitOpsSupportedForVersion(appID string, sequence int64) (bool, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return false, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return false, errors.Wrap(err, "failed to create kubernetes clientset")
	}

	_, err = clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
	if err == nil {
		// gitops secret exists -> gitops is supported
		return true, nil
	}

	query := `select kots_license from app_version where app_id = $1 and sequence = $2`
	row := s.connection.DB.QueryRow(query, appID, sequence)

	var licenseStr sql.NullString
	if err := row.Scan(&licenseStr); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to scan")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(licenseStr.String), nil, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to decode license yaml")
	}
	license := obj.(*kotsv1beta1.License)

	return license.Spec.IsGitOpsSupported, nil
}

func (s OCIStore) IsRollbackSupportedForVersion(appID string, sequence int64) (bool, error) {
	query := `select kots_app_spec from app_version where app_id = $1 and sequence = $2`
	row := s.connection.DB.QueryRow(query, appID, sequence)

	var kotsAppSpecStr sql.NullString
	if err := row.Scan(&kotsAppSpecStr); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to scan")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(kotsAppSpecStr.String), nil, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to decode kots app spec yaml")
	}
	kotsAppSpec := obj.(*kotsv1beta1.Application)

	return kotsAppSpec.Spec.AllowRollback, nil
}

func (s OCIStore) IsSnapshotsSupportedForVersion(a *apptypes.App, sequence int64) (bool, error) {
	query := `select backup_spec from app_version where app_id = $1 and sequence = $2`
	row := s.connection.DB.QueryRow(query, a.ID, sequence)

	var backupSpecStr sql.NullString
	if err := row.Scan(&backupSpecStr); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to scan")
	}

	if backupSpecStr.String == "" {
		return false, nil
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return false, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	err = s.GetAppVersionArchive(a.ID, sequence, archiveDir)
	if err != nil {
		return false, errors.Wrap(err, "failed to get app version archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return false, errors.Wrap(err, "failed to load kots kinds from path")
	}

	registrySettings, err := s.GetRegistryDetailsForApp(a.ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get registry settings for app")
	}

	// as far as I can tell, this is the only place within kotsadm/pkg/store that uses templating
	rendered, err := render.RenderFile(kotsKinds, registrySettings, sequence, a.IsAirgap, []byte(backupSpecStr.String))
	if err != nil {
		return false, errors.Wrap(err, "failed to render backup spec")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(rendered, nil, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to decode rendered backup spec yaml")
	}
	backupSpec := obj.(*velerov1.Backup)

	annotations := backupSpec.ObjectMeta.Annotations
	if annotations == nil {
		// Backup exists and there are no annotation overrides so snapshots are enabled
		return true, nil
	}

	if exclude, ok := annotations["kots.io/exclude"]; ok && exclude == "true" {
		return false, nil
	}

	if when, ok := annotations["kots.io/when"]; ok && when == "false" {
		return false, nil
	}

	return true, nil
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
func (s OCIStore) GetAppVersionArchive(appID string, sequence int64, dstPath string) error {
	// too noisy
	// logger.Debug("getting app version archive",
	// 	zap.String("appID", appID),
	// 	zap.Int64("sequence", sequence))

	storageBaseURI := os.Getenv("STORAGE_BASEURI")
	if storageBaseURI == "" {
		// KOTS 1.15 and earlier only supported s3 and there was no configuration
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

func (s OCIStore) CreateAppVersion(appID string, currentSequence *int64, appName string, appIcon string, kotsKinds *kotsutil.KotsKinds, filesInDir string, gitops gitopstypes.DownstreamGitOps, source string, skipPreflights bool) (int64, error) {
	tx, err := s.connection.DB.Begin()
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to begin")
	}
	defer tx.Rollback()

	newSequence, err := s.createAppVersion(tx, appID, currentSequence, appName, appIcon, kotsKinds)
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

	registryInfo, err := s.GetRegistryDetailsForApp(appID)
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
		licenseSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "License")
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to marshal license spec")
		}
		configSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Config")
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to marshal config spec")
		}
		configValuesSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to marshal configvalues spec")
		}

		configOpts := kotsconfig.ConfigOptions{
			ConfigSpec:       configSpec,
			ConfigValuesSpec: configValuesSpec,
			LicenseSpec:      licenseSpec,
		}
		if registryInfo != nil {
			configOpts.RegistryHost = registryInfo.Hostname
			configOpts.RegistryNamespace = registryInfo.Namespace
			configOpts.RegistryUser = registryInfo.Username
			configOpts.RegistryPassword = registryInfo.Password
		}

		// check if version needs additional configuration
		needsConfig, err := kotsconfig.NeedsConfiguration(configOpts)
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to check if version needs configuration")
		}

		downstreamStatus := "pending"
		if needsConfig {
			downstreamStatus = "pending_config"
		} else if kotsKinds.Preflight != nil && !skipPreflights {
			downstreamStatus = "pending_preflight"
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

		err = s.addAppVersionToDownstream(tx, appID, d.ClusterID, newSequence,
			kotsKinds.Installation.Spec.VersionLabel, downstreamStatus, source,
			diffSummary, diffSummaryError, commitURL, commitURL != "")
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to create downstream version")
		}
	}

	if err := tx.Commit(); err != nil {
		return int64(0), errors.Wrap(err, "failed to commit transaction")
	}

	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return int64(0), errors.Wrap(err, "failed to commit")
	}

	return newSequence, nil

}

func (s OCIStore) createAppVersion(tx *sql.Tx, appID string, currentSequence *int64, appName string, appIcon string, kotsKinds *kotsutil.KotsKinds) (int64, error) {
	// we marshal these here because it's a decision of the store to cache them in the app version table
	// not all stores will do this
	supportBundleSpec, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Collector")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal support bundle spec")
	}
	analyzersSpec, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Analyzer")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal analyzer spec")
	}
	preflightSpec, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Preflight")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal preflight spec")
	}

	appSpec, err := kotsKinds.Marshal("app.k8s.io", "v1beta1", "Application")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal app spec")
	}
	kotsAppSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Application")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal kots app spec")
	}
	kotsInstallationSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Installation")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal kots installation spec")
	}
	backupSpec, err := kotsKinds.Marshal("velero.io", "v1", "Backup")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal backup spec")
	}

	licenseSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "License")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal license spec")
	}
	configSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Config")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal config spec")
	}
	configValuesSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal configvalues spec")
	}

	newSequence := int64(0)
	if currentSequence != nil {
		row := tx.QueryRow(`select max(sequence) from app_version where app_id = $1`, appID)
		if err := row.Scan(&newSequence); err != nil {
			return 0, errors.Wrap(err, "failed to find current max sequence in row")
		}
		newSequence++
	}

	var releasedAt *time.Time
	if kotsKinds.Installation.Spec.ReleasedAt != nil {
		releasedAt = &kotsKinds.Installation.Spec.ReleasedAt.Time
	}
	query := `insert into app_version (app_id, sequence, created_at, version_label, release_notes, update_cursor, channel_id, channel_name, upstream_released_at, encryption_key,
		supportbundle_spec, analyzer_spec, preflight_spec, app_spec, kots_app_spec, kots_installation_spec, kots_license, config_spec, config_values, backup_spec)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT(app_id, sequence) DO UPDATE SET
		created_at = EXCLUDED.created_at,
		version_label = EXCLUDED.version_label,
		release_notes = EXCLUDED.release_notes,
		update_cursor = EXCLUDED.update_cursor,
		channel_id = EXCLUDED.channel_id,
		channel_name = EXCLUDED.channel_name,
		upstream_released_at = EXCLUDED.upstream_released_at,
		encryption_key = EXCLUDED.encryption_key,
		supportbundle_spec = EXCLUDED.supportbundle_spec,
		analyzer_spec = EXCLUDED.analyzer_spec,
		preflight_spec = EXCLUDED.preflight_spec,
		app_spec = EXCLUDED.app_spec,
		kots_app_spec = EXCLUDED.kots_app_spec,
		kots_installation_spec = EXCLUDED.kots_installation_spec,
		kots_license = EXCLUDED.kots_license,
		config_spec = EXCLUDED.config_spec,
		config_values = EXCLUDED.config_values,
		backup_spec = EXCLUDED.backup_spec`
	_, err = tx.Exec(query, appID, newSequence, time.Now(),
		kotsKinds.Installation.Spec.VersionLabel,
		kotsKinds.Installation.Spec.ReleaseNotes,
		kotsKinds.Installation.Spec.UpdateCursor,
		kotsKinds.Installation.Spec.ChannelID,
		kotsKinds.Installation.Spec.ChannelName,
		releasedAt,
		kotsKinds.Installation.Spec.EncryptionKey,
		supportBundleSpec,
		analyzersSpec,
		preflightSpec,
		appSpec,
		kotsAppSpec,
		kotsInstallationSpec,
		licenseSpec,
		configSpec,
		configValuesSpec,
		backupSpec)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to insert app version")
	}

	query = "update app set current_sequence = $1, name = $2, icon_uri = $3 where id = $4"
	_, err = tx.Exec(query, int64(newSequence), appName, appIcon, appID)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to update app")
	}

	return int64(newSequence), nil
}

func (s OCIStore) addAppVersionToDownstream(tx *sql.Tx, appID string, clusterID string, sequence int64, versionLabel string, status string, source string, diffSummary string, diffSummaryError string, commitURL string, gitDeployable bool) error {
	query := `insert into app_downstream_version (app_id, cluster_id, sequence, parent_sequence, created_at, version_label, status, source, diff_summary, diff_summary_error, git_commit_url, git_deployable) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := tx.Exec(
		query,
		appID,
		clusterID,
		sequence,
		sequence,
		time.Now(),
		versionLabel,
		status,
		source,
		diffSummary,
		diffSummaryError,
		commitURL,
		gitDeployable)
	if err != nil {
		return errors.Wrap(err, "failed to execute query")
	}

	return nil
}

func (s OCIStore) GetAppVersion(appID string, sequence int64) (*versiontypes.AppVersion, error) {
	query := `select sequence, created_at, status, applied_at, kots_installation_spec, kots_app_spec from app_version where app_id = $1 and sequence = $2`
	row := s.connection.DB.QueryRow(query, appID, sequence)

	var status sql.NullString
	var deployedAt sql.NullTime
	var installationSpec sql.NullString
	var kotsAppSpec sql.NullString

	v := versiontypes.AppVersion{}
	if err := row.Scan(&v.Sequence, &v.CreatedOn, &status, &deployedAt, &installationSpec, &kotsAppSpec); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, errors.Wrap(err, "failed to scan")
	}

	kotsKinds := kotsutil.KotsKinds{}

	// why is this a nullstring but we don't check if it's null?
	installation, err := kotsutil.LoadInstallationFromContents([]byte(installationSpec.String))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read installation spec")
	}
	kotsKinds.Installation = *installation

	if kotsAppSpec.Valid {
		kotsApp, err := kotsutil.LoadKotsAppFromContents([]byte(kotsAppSpec.String))
		if err != nil {
			return nil, errors.Wrap(err, "failed to read kotsapp spec")
		}
		if kotsApp != nil {
			kotsKinds.KotsApplication = *kotsApp
		}
	}

	if deployedAt.Valid {
		v.DeployedAt = &deployedAt.Time
	}

	v.KOTSKinds = &kotsKinds
	v.Status = status.String

	return &v, nil
}

func (s OCIStore) GetAppVersionsAfter(appID string, sequence int64) ([]*versiontypes.AppVersion, error) {
	query := `select sequence, created_at, status, applied_at, kots_installation_spec from app_version where app_id = $1 and sequence > $2`
	rows, err := s.connection.DB.Query(query, appID, sequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()

	var status sql.NullString
	var deployedAt sql.NullTime
	var installationSpec sql.NullString

	versions := []*versiontypes.AppVersion{}

	for rows.Next() {
		v := versiontypes.AppVersion{}
		if err := rows.Scan(&v.Sequence, &v.CreatedOn, &status, &deployedAt, &installationSpec); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		kotsKinds := kotsutil.KotsKinds{}

		installation, err := kotsutil.LoadInstallationFromContents([]byte(installationSpec.String))
		if err != nil {
			return nil, errors.Wrap(err, "failed to read installation spec")
		}
		kotsKinds.Installation = *installation

		if deployedAt.Valid {
			v.DeployedAt = &deployedAt.Time
		}

		v.Status = status.String

		versions = append(versions, &v)
	}

	return versions, nil
}

func refFromAppVersion(appID string, sequence int64, baseURI string) string {
	baseURI = strings.TrimSuffix(baseURI, "/")

	// docker images don't allow a large charset
	// so this names it registry.host/base/lower(app-id):sequence
	ref := fmt.Sprintf("%s/%s:%d", strings.TrimPrefix(baseURI, "docker://"), strings.ToLower(appID), sequence)

	return ref
}
