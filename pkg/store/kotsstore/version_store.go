package kotsstore

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/blang/semver"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	versiontypes "github.com/replicatedhq/kots/pkg/api/version/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/filestore"
	gitopstypes "github.com/replicatedhq/kots/pkg/gitops/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	kotsadmconfig "github.com/replicatedhq/kots/pkg/kotsadmconfig"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kustomize"
	"github.com/replicatedhq/kots/pkg/persistence"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	"github.com/replicatedhq/kots/pkg/secrets"
	"github.com/replicatedhq/kots/pkg/store/types"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func (s *KOTSStore) ensureApplicationMetadata(applicationMetadata string, namespace string, upstreamURI string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	existingConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "kotsadm-application-metadata", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing metadata config map")
		}

		metadata := []byte(applicationMetadata)
		_, err := clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), kotsadmobjects.ApplicationMetadataConfig(metadata, namespace, upstreamURI), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create metadata config map")
		}

		return nil
	}

	if existingConfigMap.Data == nil {
		existingConfigMap.Data = map[string]string{}
	}

	existingConfigMap.Data["application.yaml"] = applicationMetadata

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.Background(), existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}

func (s *KOTSStore) IsRollbackSupportedForVersion(appID string, sequence int64) (bool, error) {
	db := persistence.MustGetDBSession()
	query := `select kots_app_spec from app_version where app_id = $1 and sequence = $2`
	row := db.QueryRow(query, appID, sequence)

	var kotsAppSpecStr sql.NullString
	if err := row.Scan(&kotsAppSpecStr); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to scan")
	}

	if kotsAppSpecStr.String == "" {
		return false, nil
	}

	kotsAppSpec, err := kotsutil.LoadKotsAppFromContents([]byte(kotsAppSpecStr.String))
	if err != nil {
		return false, errors.Wrap(err, "failed to load kots app from contents")
	}

	return kotsAppSpec.Spec.AllowRollback, nil
}

func (s *KOTSStore) IsIdentityServiceSupportedForVersion(appID string, sequence int64) (bool, error) {
	db := persistence.MustGetDBSession()
	query := `select identity_spec from app_version where app_id = $1 and sequence = $2`
	row := db.QueryRow(query, appID, sequence)

	var identitySpecStr sql.NullString
	if err := row.Scan(&identitySpecStr); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to scan")
	}

	return identitySpecStr.String != "", nil
}

func (s *KOTSStore) IsSnapshotsSupportedForVersion(a *apptypes.App, sequence int64, renderer rendertypes.Renderer) (bool, error) {
	db := persistence.MustGetDBSession()
	query := `select backup_spec from app_version where app_id = $1 and sequence = $2`
	row := db.QueryRow(query, a.ID, sequence)

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

	// as far as I can tell, this is the only place within pkg/store that uses templating
	rendered, err := renderer.RenderFile(kotsKinds, registrySettings, a.Slug, sequence, a.IsAirgap, util.PodNamespace, []byte(backupSpecStr.String))
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

func (s *KOTSStore) GetTargetKotsVersionForVersion(appID string, sequence int64) (string, error) {
	db := persistence.MustGetDBSession()
	query := `select kots_app_spec from app_version where app_id = $1 and sequence = $2`
	row := db.QueryRow(query, appID, sequence)

	var kotsAppSpecStr sql.NullString
	if err := row.Scan(&kotsAppSpecStr); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", errors.Wrap(err, "failed to scan")
	}

	if kotsAppSpecStr.String == "" {
		return "", nil
	}

	kotsAppSpec, err := kotsutil.LoadKotsAppFromContents([]byte(kotsAppSpecStr.String))
	if err != nil {
		return "", errors.Wrap(err, "failed to load kots app from contents")
	}

	return kotsAppSpec.Spec.TargetKotsVersion, nil
}

// CreateAppVersion takes an unarchived app, makes an archive and then uploads it
// to s3 with the appID and sequence specified
func (s *KOTSStore) CreateAppVersionArchive(appID string, sequence int64, archivePath string) error {
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

	f, err := os.Open(fileToUpload)
	if err != nil {
		return errors.Wrap(err, "failed to open archive file")
	}
	defer f.Close()

	outputPath := fmt.Sprintf("%s/%d.tar.gz", appID, sequence)
	err = filestore.GetStore().WriteArchive(outputPath, f)
	if err != nil {
		return errors.Wrap(err, "failed to write archive")
	}

	return nil
}

// GetAppVersionArchive will fetch the archive and extract it into the given dstPath directory name
func (s *KOTSStore) GetAppVersionArchive(appID string, sequence int64, dstPath string) error {
	// too noisy
	// logger.Debug("getting app version archive",
	// 	zap.String("appID", appID),
	// 	zap.Int64("sequence", sequence))

	path := fmt.Sprintf("%s/%d.tar.gz", appID, sequence)
	bundlePath, err := filestore.GetStore().ReadArchive(path)
	if err != nil {
		return errors.Wrap(err, "failed to read archive")
	}
	defer os.RemoveAll(bundlePath)

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(bundlePath, dstPath); err != nil {
		return errors.Wrap(err, "failed to unarchive")
	}

	return nil
}

// GetAppVersionBaseSequence returns the base sequence for a given version label.
// if the "versionLabel" param is empty or is not a valid semver, the sequence of the latest version will be returned.
func (s *KOTSStore) GetAppVersionBaseSequence(appID string, versionLabel string) (int64, error) {
	appVersions, err := s.FindAppVersions(appID, true)
	if err != nil {
		return -1, errors.Wrapf(err, "failed to find app versions for app %s", appID)
	}

	mockVersion := &downstreamtypes.DownstreamVersion{
		// to id the mocked version and be able to retrieve it later.
		// use "MaxInt64" so that it ends up on the top of the list if it's not a semvered version.
		Sequence: math.MaxInt64,
	}

	targetSemver, err := semver.ParseTolerant(versionLabel)
	if err == nil {
		mockVersion.Semver = &targetSemver
	}

	license, err := s.GetLatestLicenseForApp(appID)
	if err != nil {
		return -1, errors.Wrap(err, "failed to get app license")
	}

	// add to the top of the list and sort
	appVersions.AllVersions = append([]*downstreamtypes.DownstreamVersion{mockVersion}, appVersions.AllVersions...)
	downstreamtypes.SortDownstreamVersions(appVersions, license.Spec.IsSemverRequired)

	var baseVersion *downstreamtypes.DownstreamVersion
	for i, v := range appVersions.AllVersions {
		if v.Sequence == math.MaxInt64 {
			// this is our mocked version, base it off of the previous version in the sorted list (if exists).
			if i < len(appVersions.AllVersions)-1 {
				baseVersion = appVersions.AllVersions[i+1]
			}
			// remove the mocked version from the list to not affect what the latest version is in case there's no previous version to use as base.
			appVersions.AllVersions = append(appVersions.AllVersions[:i], appVersions.AllVersions[i+1:]...)
			break
		}
	}

	// if a previous version was not found, base off of the latest version
	if baseVersion == nil {
		baseVersion = appVersions.AllVersions[0]
	}

	return baseVersion.ParentSequence, nil
}

// GetAppVersionBaseArchive returns the base archive directory for a given version label.
// if the "versionLabel" param is empty or is not a valid semver, the archive of the latest version will be returned.
// the base archive directory contains data such as config values.
// caller is responsible for cleaning up the created archive dir.
// returns the path to the archive and the base sequence.
func (s *KOTSStore) GetAppVersionBaseArchive(appID string, versionLabel string) (string, int64, error) {
	baseSequence, err := s.GetAppVersionBaseSequence(appID, versionLabel)
	if err != nil {
		return "", -1, errors.Wrapf(err, "failed to get base sequence for version %s", versionLabel)
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", -1, errors.Wrap(err, "failed to create temp dir")
	}

	err = s.GetAppVersionArchive(appID, baseSequence, archiveDir)
	if err != nil {
		return "", -1, errors.Wrap(err, "failed to get app version archive")
	}

	return archiveDir, baseSequence, nil
}

func (s *KOTSStore) CreatePendingDownloadAppVersion(appID string, update upstreamtypes.Update, kotsApplication *kotsv1beta1.Application, license *kotsv1beta1.License) (int64, error) {
	db := persistence.MustGetDBSession()

	tx, err := db.Begin()
	if err != nil {
		return 0, errors.Wrap(err, "failed to begin")
	}
	defer tx.Rollback()

	a, err := s.GetApp(appID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get app")
	}

	kotsKinds := kotsutil.EmptyKotsKinds()
	if kotsApplication != nil {
		kotsKinds.KotsApplication = *kotsApplication
	}
	kotsKinds.License = license

	var releasedAt *metav1.Time
	if update.ReleasedAt != nil {
		releasedAt = &metav1.Time{Time: *update.ReleasedAt}
	}
	kotsKinds.Installation = kotsv1beta1.Installation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "Installation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", update.Cursor, update.VersionLabel),
		},
		Spec: kotsv1beta1.InstallationSpec{
			UpdateCursor: update.Cursor,
			ChannelID:    update.ChannelID,
			ChannelName:  update.ChannelName,
			VersionLabel: update.VersionLabel,
			ReleasedAt:   releasedAt,
			ReleaseNotes: update.ReleaseNotes,
		},
	}

	newSequence, err := s.createAppVersionRecord(tx, a.ID, a.Name, a.IconURI, &kotsKinds)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create app version")
	}

	downstreams, err := s.ListDownstreamsForApp(appID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to list downstreams")
	}

	for _, d := range downstreams {
		err = s.addAppVersionToDownstream(tx, a.ID, d.ClusterID, newSequence,
			kotsKinds.Installation.Spec.VersionLabel, types.VersionPendingDownload, "Upstream Update",
			"", "", "", false, false)
		if err != nil {
			return 0, errors.Wrap(err, "failed to create downstream version")
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrap(err, "failed to commit")
	}

	return newSequence, nil
}

func (s *KOTSStore) UpdateAppVersion(appID string, sequence int64, baseSequence *int64, filesInDir string, source string, skipPreflights bool, gitops gitopstypes.DownstreamGitOps, renderer rendertypes.Renderer) error {
	// make sure version exists first
	if v, err := s.GetAppVersion(appID, sequence); err != nil {
		return errors.Wrap(err, "failed to get app version")
	} else if v == nil {
		return errors.Errorf("version %d not found", sequence)
	}

	db := persistence.MustGetDBSession()

	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin")
	}
	defer tx.Rollback()

	if err := s.upsertAppVersion(tx, appID, sequence, baseSequence, filesInDir, source, skipPreflights, gitops, renderer); err != nil {
		return errors.Wrap(err, "failed to upsert app version")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}

func (s *KOTSStore) CreateAppVersion(appID string, baseSequence *int64, filesInDir string, source string, skipPreflights bool, gitops gitopstypes.DownstreamGitOps, renderer rendertypes.Renderer) (int64, error) {
	db := persistence.MustGetDBSession()

	tx, err := db.Begin()
	if err != nil {
		return 0, errors.Wrap(err, "failed to begin")
	}
	defer tx.Rollback()

	newSequence, err := s.createAppVersion(tx, appID, baseSequence, filesInDir, source, skipPreflights, gitops, renderer)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrap(err, "failed to commit")
	}

	return newSequence, nil
}

func (s *KOTSStore) createAppVersion(tx *sql.Tx, appID string, baseSequence *int64, filesInDir string, source string, skipPreflights bool, gitops gitopstypes.DownstreamGitOps, renderer rendertypes.Renderer) (int64, error) {
	newSequence, err := s.getNextAppSequence(tx, appID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get next sequence number")
	}

	if err := s.upsertAppVersion(tx, appID, newSequence, baseSequence, filesInDir, source, skipPreflights, gitops, renderer); err != nil {
		return 0, errors.Wrap(err, "failed to upsert app version")
	}

	return newSequence, nil
}

func (s *KOTSStore) upsertAppVersion(tx *sql.Tx, appID string, sequence int64, baseSequence *int64, filesInDir string, source string, skipPreflights bool, gitops gitopstypes.DownstreamGitOps, renderer rendertypes.Renderer) error {
	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(filesInDir)
	if err != nil {
		return errors.Wrap(err, "failed to read kots kinds")
	}

	appName := kotsKinds.KotsApplication.Spec.Title
	a, err := s.GetApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}
	if appName == "" {
		appName = a.Name
	}

	appIcon := kotsKinds.KotsApplication.Spec.Icon

	renderedPreflight, err := s.renderPreflightSpec(appID, a.Slug, sequence, a.IsAirgap, kotsKinds, renderer)
	if err != nil {
		return errors.Wrap(err, "failed to render app preflight spec")
	}
	kotsKinds.Preflight = renderedPreflight

	if err := s.upsertAppVersionRecord(tx, appID, sequence, appName, appIcon, kotsKinds); err != nil {
		return errors.Wrap(err, "failed to upsert app version record")
	}

	if err := secrets.ReplaceSecretsInPath(filesInDir); err != nil {
		return errors.Wrap(err, "failed to replace secrets")
	}
	if err := s.CreateAppVersionArchive(appID, sequence, filesInDir); err != nil {
		return errors.Wrap(err, "failed to create app version archive")
	}

	previousArchiveDir := ""
	if baseSequence != nil {
		previousDir, err := ioutil.TempDir("", "kotsadm")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(previousDir)

		// Get the previous archive, we need this to calculate the diff
		err = s.GetAppVersionArchive(appID, *baseSequence, previousDir)
		if err != nil {
			return errors.Wrap(err, "failed to get previous archive")
		}

		previousArchiveDir = previousDir
	}

	registrySettings, err := s.GetRegistryDetailsForApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app registry info")
	}

	downstreams, err := s.ListDownstreamsForApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to list downstreams")
	}

	kustomizeBinPath := kotsKinds.GetKustomizeBinaryPath()

	for _, d := range downstreams {
		// there's a small chance this is not optimal, but no current code path
		// will support multiple downstreams, so this is cleaner here for now

		hasStrictPreflights, err := kotsutil.HasStrictPreflights(renderedPreflight)
		if err != nil {
			return errors.Wrap(err, "failed to check strict preflights from spec")
		}
		downstreamStatus := types.VersionPending
		if baseSequence == nil && kotsKinds.IsConfigurable() { // initial version should always require configuration (if exists) even if all required items are already set and have values (except for automated installs, which can override this later)
			downstreamStatus = types.VersionPendingConfig
		} else if kotsKinds.HasPreflights() && (!skipPreflights || hasStrictPreflights) {
			downstreamStatus = types.VersionPendingPreflight
		}
		if baseSequence != nil { // only check if the version needs configuration for later versions (not the initial one) since the config is always required for the initial version (except for automated installs, which can override that later)
			// check if version needs additional configuration
			t, err := kotsadmconfig.NeedsConfiguration(a.Slug, sequence, a.IsAirgap, kotsKinds, registrySettings)
			if err != nil {
				return errors.Wrap(err, "failed to check if version needs configuration")
			}
			if t {
				downstreamStatus = types.VersionPendingConfig
			}
		}

		diffSummary, diffSummaryError := "", ""
		if baseSequence != nil {
			// diff this release from the last release
			diff, err := kustomize.DiffAppVersionsForDownstream(d.Name, filesInDir, previousArchiveDir, kustomizeBinPath)
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

		commitURL, err := gitops.CreateGitOpsDownstreamCommit(appID, d.ClusterID, int(sequence), filesInDir, d.Name)
		if err != nil {
			return errors.Wrap(err, "failed to create gitops commit")
		}

		err = s.addAppVersionToDownstream(tx, appID, d.ClusterID, sequence,
			kotsKinds.Installation.Spec.VersionLabel, downstreamStatus, source,
			diffSummary, diffSummaryError, commitURL, commitURL != "", skipPreflights)
		if err != nil {
			return errors.Wrap(err, "failed to create downstream version")
		}

		// update metadata configmap
		applicationSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Application")
		if err != nil {
			return errors.Wrap(err, "failed to marshal application spec")
		}

		if err := s.ensureApplicationMetadata(applicationSpec, util.PodNamespace, a.UpstreamURI); err != nil {
			return errors.Wrap(err, "failed to get metadata config map")
		}
	}

	return nil
}

func (s *KOTSStore) createAppVersionRecord(tx *sql.Tx, appID string, appName string, appIcon string, kotsKinds *kotsutil.KotsKinds) (int64, error) {
	newSequence, err := s.getNextAppSequence(tx, appID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get next sequence number")
	}

	if err := s.upsertAppVersionRecord(tx, appID, newSequence, appName, appIcon, kotsKinds); err != nil {
		return 0, errors.Wrap(err, "failed to upsert app version record")
	}

	return newSequence, nil
}

func (s *KOTSStore) upsertAppVersionRecord(tx *sql.Tx, appID string, sequence int64, appName string, appIcon string, kotsKinds *kotsutil.KotsKinds) error {
	// we marshal these here because it's a decision of the store to cache them in the app version table
	// not all stores will do this
	supportBundleSpec, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Collector")
	if err != nil {
		return errors.Wrap(err, "failed to marshal support bundle spec")
	}
	analyzersSpec, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Analyzer")
	if err != nil {
		return errors.Wrap(err, "failed to marshal analyzer spec")
	}
	preflightSpec, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Preflight")
	if err != nil {
		return errors.Wrap(err, "failed to marshal preflight spec")
	}

	appSpec, err := kotsKinds.Marshal("app.k8s.io", "v1beta1", "Application")
	if err != nil {
		return errors.Wrap(err, "failed to marshal app spec")
	}
	kotsAppSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Application")
	if err != nil {
		return errors.Wrap(err, "failed to marshal kots app spec")
	}
	kotsInstallationSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Installation")
	if err != nil {
		return errors.Wrap(err, "failed to marshal kots installation spec")
	}

	backupSpec, err := kotsKinds.Marshal("velero.io", "v1", "Backup")
	if err != nil {
		return errors.Wrap(err, "failed to marshal backup spec")
	}
	identitySpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Identity")
	if err != nil {
		return errors.Wrap(err, "failed to marshal identity spec")
	}

	licenseSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "License")
	if err != nil {
		return errors.Wrap(err, "failed to marshal license spec")
	}
	configSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Config")
	if err != nil {
		return errors.Wrap(err, "failed to marshal config spec")
	}
	configValuesSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
	if err != nil {
		return errors.Wrap(err, "failed to marshal configvalues spec")
	}

	var releasedAt *time.Time
	if kotsKinds.Installation.Spec.ReleasedAt != nil {
		releasedAt = &kotsKinds.Installation.Spec.ReleasedAt.Time
	}
	query := `insert into app_version (app_id, sequence, created_at, version_label, release_notes, update_cursor, channel_id, channel_name, upstream_released_at, encryption_key,
		supportbundle_spec, analyzer_spec, preflight_spec, app_spec, kots_app_spec, kots_installation_spec, kots_license, config_spec, config_values, backup_spec, identity_spec)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
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
		backup_spec = EXCLUDED.backup_spec,
		identity_spec = EXCLUDED.identity_spec`
	_, err = tx.Exec(query, appID, sequence, time.Now(),
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
		backupSpec,
		identitySpec)
	if err != nil {
		return errors.Wrap(err, "failed to insert app version")
	}

	// an old version could be downloaded at a later point, pick higher sequence
	query = "update app set current_sequence = greatest(current_sequence, $1), name = $2, icon_uri = $3 where id = $4"
	_, err = tx.Exec(query, sequence, appName, appIcon, appID)
	if err != nil {
		return errors.Wrap(err, "failed to update app")
	}

	return nil
}

func (s *KOTSStore) addAppVersionToDownstream(tx *sql.Tx, appID string, clusterID string, sequence int64, versionLabel string, status types.DownstreamVersionStatus, source string, diffSummary string, diffSummaryError string, commitURL string, gitDeployable bool, preflightsSkipped bool) error {
	query := `insert into app_downstream_version (app_id, cluster_id, sequence, parent_sequence, created_at, version_label, status, source, diff_summary, diff_summary_error, git_commit_url, git_deployable, preflight_skipped)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT(app_id, cluster_id, sequence) DO UPDATE SET
		created_at = EXCLUDED.created_at,
		version_label = EXCLUDED.version_label,
		status = EXCLUDED.status,
		source = EXCLUDED.source,
		diff_summary = EXCLUDED.diff_summary,
		diff_summary_error = EXCLUDED.diff_summary_error,
		git_commit_url = EXCLUDED.git_commit_url,
		git_deployable = EXCLUDED.git_deployable,
		preflight_skipped= EXCLUDED.preflight_skipped`
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
		gitDeployable,
		preflightsSkipped)
	if err != nil {
		return errors.Wrap(err, "failed to execute query")
	}

	return nil
}

func (s *KOTSStore) GetAppVersion(appID string, sequence int64) (*versiontypes.AppVersion, error) {
	db := persistence.MustGetDBSession()
	query := `select sequence, created_at, status, applied_at, kots_installation_spec, kots_app_spec, kots_license from app_version where app_id = $1 and sequence = $2`
	row := db.QueryRow(query, appID, sequence)

	var status sql.NullString
	var deployedAt persistence.NullStringTime
	var createdAt persistence.NullStringTime
	var installationSpec sql.NullString
	var kotsAppSpec sql.NullString
	var licenseSpec sql.NullString

	v := versiontypes.AppVersion{
		AppID: appID,
	}
	if err := row.Scan(&v.Sequence, &createdAt, &status, &createdAt, &installationSpec, &kotsAppSpec, &licenseSpec); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, errors.Wrap(err, "failed to scan")
	}

	v.KOTSKinds = &kotsutil.KotsKinds{}

	if installationSpec.Valid && installationSpec.String != "" {
		installation, err := kotsutil.LoadInstallationFromContents([]byte(installationSpec.String))
		if err != nil {
			return nil, errors.Wrap(err, "failed to read installation spec")
		}
		if installation != nil {
			v.KOTSKinds.Installation = *installation
		}
	}

	if kotsAppSpec.Valid && kotsAppSpec.String != "" {
		kotsApp, err := kotsutil.LoadKotsAppFromContents([]byte(kotsAppSpec.String))
		if err != nil {
			return nil, errors.Wrap(err, "failed to read kotsapp spec")
		}
		if kotsApp != nil {
			v.KOTSKinds.KotsApplication = *kotsApp
		}
	}

	if licenseSpec.Valid && licenseSpec.String != "" {
		license, err := kotsutil.LoadLicenseFromBytes([]byte(licenseSpec.String))
		if err != nil {
			return nil, errors.Wrap(err, "failed to read license spec")
		}
		if license != nil {
			v.KOTSKinds.License = license
		}
	}

	v.CreatedOn = createdAt.Time
	if deployedAt.Valid {
		v.DeployedAt = &deployedAt.Time
	}

	v.Status = status.String

	return &v, nil
}

func (s *KOTSStore) GetLatestAppVersion(appID string, downloadedOnly bool) (*versiontypes.AppVersion, error) {
	downstreamVersions, err := s.FindAppVersions(appID, downloadedOnly)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find app versions")
	}
	if len(downstreamVersions.AllVersions) == 0 {
		return nil, errors.New("no app versions found")
	}
	return s.GetAppVersion(appID, downstreamVersions.AllVersions[0].ParentSequence)
}

func (s *KOTSStore) UpdateNextAppVersionDiffSummary(appID string, baseSequence int64) error {
	a, err := s.GetApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	appVersions, err := s.FindAppVersions(a.ID, true)
	if err != nil {
		return errors.Wrapf(err, "failed to find app versions for app %s", appID)
	}

	nextSequence := int64(-1)
	for _, v := range appVersions.AllVersions {
		if v.ParentSequence == baseSequence {
			break
		}
		nextSequence = v.ParentSequence
	}

	if nextSequence == -1 {
		return nil
	}

	downstreams, err := s.ListDownstreamsForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to list downstreams")
	}
	if len(downstreams) == 0 {
		return errors.Errorf("no downstreams found for app %q", a.Slug)
	}
	d := downstreams[0]

	baseArchiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create base temp dir")
	}
	defer os.RemoveAll(baseArchiveDir)

	if err := s.GetAppVersionArchive(appID, baseSequence, baseArchiveDir); err != nil {
		return errors.Wrap(err, "failed to get base archive dir")
	}

	nextArchiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create next temp dir")
	}
	defer os.RemoveAll(nextArchiveDir)

	if err := s.GetAppVersionArchive(appID, nextSequence, nextArchiveDir); err != nil {
		return errors.Wrap(err, "failed to get next archive dir")
	}

	nextKotsKinds, err := kotsutil.LoadKotsKindsFromPath(nextArchiveDir)
	if err != nil {
		return errors.Wrap(err, "failed to read kots kinds")
	}

	diffSummary, diffSummaryError := "", ""

	diff, err := kustomize.DiffAppVersionsForDownstream(d.Name, nextArchiveDir, baseArchiveDir, nextKotsKinds.GetKustomizeBinaryPath())
	if err != nil {
		diffSummaryError = errors.Wrap(err, "failed to diff").Error()
	} else {
		b, err := json.Marshal(diff)
		if err != nil {
			diffSummaryError = errors.Wrap(err, "failed to marshal diff").Error()
		}
		diffSummary = string(b)
	}

	db := persistence.MustGetDBSession()
	query := `update app_downstream_version set diff_summary = $3, diff_summary_error = $4 where app_id = $1 AND sequence = $2`
	if _, err := db.Exec(query, appID, nextSequence, diffSummary, diffSummaryError); err != nil {
		return errors.Wrap(err, "failed to execute query")
	}

	return nil
}

func (s *KOTSStore) UpdateAppVersionInstallationSpec(appID string, sequence int64, installation kotsv1beta1.Installation) error {
	ser := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := ser.Encode(&installation, &b); err != nil {
		return errors.Wrap(err, "failed to encode installation")
	}

	db := persistence.MustGetDBSession()
	query := `UPDATE app_version SET kots_installation_spec = $1 WHERE app_id = $2 AND sequence = $3`
	_, err := db.Exec(query, b.String(), appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s *KOTSStore) GetNextAppSequence(appID string) (int64, error) {
	return s.getNextAppSequence(persistence.MustGetDBSession(), appID)
}

func (s *KOTSStore) getNextAppSequence(db queryable, appID string) (int64, error) {
	var maxSequence sql.NullInt64
	row := db.QueryRow(`select max(sequence) from app_version where app_id = $1`, appID)
	if err := row.Scan(&maxSequence); err != nil {
		return 0, errors.Wrap(err, "failed to find current max sequence in row")
	}

	newSequence := int64(0)
	if maxSequence.Valid {
		newSequence = maxSequence.Int64 + 1
	}

	return newSequence, nil
}

func (s *KOTSStore) GetCurrentUpdateCursor(appID string, channelID string) (string, string, error) {
	db := persistence.MustGetDBSession()
	query := `SELECT update_cursor, version_label FROM app_version WHERE app_id = $1 AND channel_id = $2 AND update_cursor::INT IN (
		SELECT MAX(update_cursor::INT) FROM app_version WHERE app_id = $1 AND channel_id = $2
	) ORDER BY sequence DESC LIMIT 1`
	row := db.QueryRow(query, appID, channelID)

	var updateCursor sql.NullString
	var versionLabel sql.NullString

	if err := row.Scan(&updateCursor, &versionLabel); err != nil {
		if err == sql.ErrNoRows {
			return "", "", nil
		}
		return "", "", errors.Wrap(err, "failed to scan")
	}

	return updateCursor.String, versionLabel.String, nil
}

func (s *KOTSStore) HasStrictPreflights(appID string, sequence int64) (bool, error) {
	var preflightSpecStr sql.NullString
	db := persistence.MustGetDBSession()
	query := `SELECT preflight_spec FROM app_version WHERE app_id = $1 AND sequence = $2`
	row := db.QueryRow(query, appID, sequence)

	if err := row.Scan(&preflightSpecStr); err != nil {
		return false, errors.Wrap(err, "failed to scan")
	}
	return s.hasStrictPreflights(preflightSpecStr)
}

func (s *KOTSStore) renderPreflightSpec(appID string, appSlug string, sequence int64, isAirgap bool, kotsKinds *kotsutil.KotsKinds, renderer rendertypes.Renderer) (*troubleshootv1beta2.Preflight, error) {
	if kotsKinds.HasPreflights() {
		// render the preflight file
		// we need to convert to bytes first, so that we can reuse the renderfile function
		renderedMarshalledPreflights, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Preflight")
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal preflight")
		}

		registrySettings, err := s.GetRegistryDetailsForApp(appID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get registry settings for app")
		}

		renderedPreflight, err := renderer.RenderFile(kotsKinds, registrySettings, appSlug, sequence, isAirgap, util.PodNamespace, []byte(renderedMarshalledPreflights))
		if err != nil {
			return nil, errors.Wrap(err, "failed to render preflights")
		}
		preflight, err := kotsutil.LoadPreflightFromContents(renderedPreflight)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load rendered preflight")
		}
		return preflight, nil
	}

	return nil, nil
}
