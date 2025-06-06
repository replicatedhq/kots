package kotsstore

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	versiontypes "github.com/replicatedhq/kots/pkg/api/version/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/apparchive"
	"github.com/replicatedhq/kots/pkg/binaries"
	"github.com/replicatedhq/kots/pkg/cursor"
	"github.com/replicatedhq/kots/pkg/filestore"
	"github.com/replicatedhq/kots/pkg/gitops"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmconfig "github.com/replicatedhq/kots/pkg/kotsadmconfig"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	"github.com/replicatedhq/kots/pkg/secrets"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/store/types"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/rqlite/gorqlite"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func (s *KOTSStore) IsRollbackSupportedForVersion(appID string, sequence int64) (bool, error) {
	db := persistence.MustGetDBSession()
	query := `select kots_app_spec from app_version where app_id = ? and sequence = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, sequence},
	})
	if err != nil {
		return false, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return false, nil
	}

	var kotsAppSpecStr gorqlite.NullString
	if err := rows.Scan(&kotsAppSpecStr); err != nil {
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
	query := `select identity_spec from app_version where app_id = ? and sequence = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, sequence},
	})
	if err != nil {
		return false, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return false, nil
	}

	var identitySpecStr gorqlite.NullString
	if err := rows.Scan(&identitySpecStr); err != nil {
		return false, errors.Wrap(err, "failed to scan")
	}

	return identitySpecStr.String != "", nil
}

func (s *KOTSStore) IsSnapshotsSupportedForVersion(a *apptypes.App, sequence int64, renderer rendertypes.Renderer) (bool, error) {
	if util.IsEmbeddedCluster() {
		return true, nil
	}

	db := persistence.MustGetDBSession()
	query := `select backup_spec from app_version where app_id = ? and sequence = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{a.ID, sequence},
	})
	if err != nil {
		return false, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return false, nil
	}

	var backupSpecStr gorqlite.NullString
	if err := rows.Scan(&backupSpecStr); err != nil {
		return false, errors.Wrap(err, "failed to scan")
	}

	if backupSpecStr.String == "" {
		return false, nil
	}

	archiveDir, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return false, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	err = s.GetAppVersionArchive(a.ID, sequence, archiveDir)
	if err != nil {
		return false, errors.Wrap(err, "failed to get app version archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
	if err != nil {
		return false, errors.Wrap(err, "failed to load kots kinds from path")
	}

	registrySettings, err := s.GetRegistryDetailsForApp(a.ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get registry settings for app")
	}

	// as far as I can tell, this is the only place within pkg/store that uses templating
	rendered, err := renderer.RenderFile(rendertypes.RenderFileOptions{
		KotsKinds:        kotsKinds,
		RegistrySettings: registrySettings,
		AppSlug:          a.Slug,
		Sequence:         sequence,
		IsAirgap:         a.IsAirgap,
		Namespace:        util.PodNamespace,
		InputContent:     []byte(backupSpecStr.String),
	})
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
	query := `select kots_app_spec from app_version where app_id = ? and sequence = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, sequence},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return "", nil
	}

	var kotsAppSpecStr gorqlite.NullString
	if err := rows.Scan(&kotsAppSpecStr); err != nil {
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

func (s *KOTSStore) GetEmbeddedClusterConfigForVersion(appID string, sequence int64) (*embeddedclusterv1beta1.Config, error) {
	db := persistence.MustGetDBSession()
	query := `select embeddedcluster_config from app_version where app_id = ? and sequence = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, sequence},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return nil, nil
	}

	var embeddedClusterSpecStr gorqlite.NullString
	if err := rows.Scan(&embeddedClusterSpecStr); err != nil {
		return nil, errors.Wrap(err, "failed to scan")
	}

	if embeddedClusterSpecStr.String == "" {
		return nil, nil
	}

	embeddedClusterConfig, err := kotsutil.LoadEmbeddedClusterConfigFromBytes([]byte(embeddedClusterSpecStr.String))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load embedded cluster config from contents")
	}

	return embeddedClusterConfig, nil
}

// GetAppVersionArchive will fetch the archive and extract it into the given dstPath directory name
func (s *KOTSStore) GetAppVersionArchive(appID string, sequence int64, dstPath string) error {
	path := fmt.Sprintf("%s/%d.tar.gz", appID, sequence)
	bundlePath, err := filestore.GetStore().ReadArchive(path)
	if err != nil {
		return errors.Wrap(err, "failed to read archive")
	}
	defer os.RemoveAll(bundlePath)

	if err := util.ExtractTGZArchive(bundlePath, dstPath); err != nil {
		return errors.Wrap(err, "failed to extract archive")
	}

	return nil
}

// GetAppVersionBaseSequence returns the base sequence for a given version label.
// if the "versionLabel" param is empty or is not a valid semver, the sequence of the latest version will be returned.
func (s *KOTSStore) GetAppVersionBaseSequence(appID string, versionLabel string) (int64, error) {
	appVersions, err := s.FindDownstreamVersions(appID, true)
	if err != nil {
		return -1, errors.Wrapf(err, "failed to find app downstream versions for app %s", appID)
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
	downstreamtypes.SortDownstreamVersions(appVersions.AllVersions, license.Spec.IsSemverRequired)

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

	archiveDir, err := os.MkdirTemp("", "kotsadm")
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
	statements := []gorqlite.ParameterizedStatement{}

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
			IsRequired:   update.IsRequired,
			ReleasedAt:   releasedAt,
			ReleaseNotes: update.ReleaseNotes,
		},
	}

	appVersionStatements, newSequence, err := s.createAppVersionRecordStatements(a.ID, a.Name, a.IconURI, &kotsKinds, nil)
	if err != nil {
		return 0, errors.Wrap(err, "failed to construct app version record statements")
	}
	statements = append(statements, appVersionStatements...)

	downstreams, err := s.ListDownstreamsForApp(appID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to list downstreams")
	}

	for _, d := range downstreams {
		downstreamVersionStatements, err := s.upsertAppDownstreamVersionStatements(a.ID, d.ClusterID, newSequence,
			kotsKinds.Installation.Spec.VersionLabel, types.VersionPendingDownload,
			"Upstream Update", "", "", "", false, false)
		if err != nil {
			return 0, errors.Wrap(err, "failed to construct app downstream version statements")
		}
		statements = append(statements, downstreamVersionStatements...)
	}

	if wrs, err := db.WriteParameterized(statements); err != nil {
		wrErrs := []error{}
		for _, wr := range wrs {
			wrErrs = append(wrErrs, wr.Err)
		}
		return 0, fmt.Errorf("failed to write: %v: %v", err, wrErrs)
	}

	return newSequence, nil
}

func (s *KOTSStore) UpdateAppVersion(appID string, sequence int64, baseSequence *int64, filesInDir string, source string, skipPreflights bool) error {
	// make sure version exists first
	if v, err := s.GetAppVersion(appID, sequence); err != nil {
		return errors.Wrap(err, "failed to get app version")
	} else if v == nil {
		return errors.Errorf("version %d not found", sequence)
	}

	db := persistence.MustGetDBSession()

	appVersionStatements, err := s.upsertAppVersionStatements(appID, sequence, baseSequence, filesInDir, source, false, false, skipPreflights)
	if err != nil {
		return errors.Wrap(err, "failed to construct app version statements")
	}

	if wrs, err := db.WriteParameterized(appVersionStatements); err != nil {
		wrErrs := []error{}
		for _, wr := range wrs {
			wrErrs = append(wrErrs, wr.Err)
		}
		return fmt.Errorf("failed to write: %v: %v", err, wrErrs)
	}

	return nil
}

func (s *KOTSStore) CreateAppVersion(appID string, baseSequence *int64, filesInDir string, source string, isInstall bool, isAutomated bool, skipPreflights bool) (int64, error) {
	db := persistence.MustGetDBSession()
	appVersionStatements, newSequence, err := s.createAppVersionStatements(appID, baseSequence, filesInDir, source, isInstall, isAutomated, skipPreflights)
	if err != nil {
		return 0, errors.Wrap(err, "failed to construct app version statements")
	}

	if wrs, err := db.WriteParameterized(appVersionStatements); err != nil {
		wrErrs := []error{}
		for _, wr := range wrs {
			wrErrs = append(wrErrs, wr.Err)
		}
		return 0, fmt.Errorf("failed to write: %v: %v", err, wrErrs)
	}

	return newSequence, nil
}

func (s *KOTSStore) createAppVersionStatements(appID string, baseSequence *int64, filesInDir string, source string, isInstall bool, isAutomated bool, skipPreflights bool) ([]gorqlite.ParameterizedStatement, int64, error) {
	newSequence, err := s.GetNextAppSequence(appID)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to get next sequence number")
	}

	appVersionStatements, err := s.upsertAppVersionStatements(appID, newSequence, baseSequence, filesInDir, source, isInstall, isAutomated, skipPreflights)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to construct app version statements")
	}

	return appVersionStatements, newSequence, nil
}

func (s *KOTSStore) upsertAppVersionStatements(appID string, sequence int64, baseSequence *int64, filesInDir string, source string, isInstall bool, isAutomated bool, skipPreflights bool) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	kotsKinds, err := kotsutil.LoadKotsKinds(filesInDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read kots kinds")
	}

	appName := kotsKinds.KotsApplication.Spec.Title
	a, err := s.GetApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}
	if appName == "" {
		appName = a.Name
	}

	appIcon := kotsKinds.KotsApplication.Spec.Icon

	brandingArchive, err := kotsutil.LoadBrandingArchiveFromPath(filepath.Join(filesInDir, "upstream"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load branding archive")
	}

	appVersionRecordStatements, err := s.upsertAppVersionRecordStatements(appID, sequence, appName, appIcon, kotsKinds, brandingArchive.Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct app version record statements")
	}
	statements = append(statements, appVersionRecordStatements...)

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	if err := secrets.ReplaceSecretsInPath(filesInDir, clientset); err != nil {
		return nil, errors.Wrap(err, "failed to replace secrets")
	}
	if err := apparchive.CreateAppVersionArchive(filesInDir, fmt.Sprintf("%s/%d.tar.gz", appID, sequence)); err != nil {
		return nil, errors.Wrap(err, "failed to create app version archive")
	}

	previousArchiveDir := ""
	if baseSequence != nil {
		previousDir, err := os.MkdirTemp("", "kotsadm")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(previousDir)

		// Get the previous archive, we need this to calculate the diff
		err = s.GetAppVersionArchive(appID, *baseSequence, previousDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get previous archive")
		}

		previousArchiveDir = previousDir
	}

	downstreams, err := s.ListDownstreamsForApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams")
	}

	kustomizeBinPath := binaries.GetKustomizeBinPath()

	for _, d := range downstreams {
		// there's a small chance this is not optimal, but no current code path
		// will support multiple downstreams, so this is cleaner here for now
		downstreamStatus, err := determineDownstreamVersionStatus(s, a, sequence, kotsKinds, isInstall, isAutomated, skipPreflights)
		if err != nil {
			return nil, errors.Wrap(err, "failed to determine downstream version status")
		}

		diffSummary, diffSummaryError := "", ""
		if baseSequence != nil {
			// diff this release from the last release
			diff, err := apparchive.DiffAppVersionsForDownstream(d.Name, filesInDir, previousArchiveDir, kustomizeBinPath)
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

		commitURL, err := gitops.CreateGitOpsDownstreamCommit(a, d.ClusterID, int(sequence), filesInDir, d.Name)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create gitops commit")
		}

		downstreamVersionStatements, err := s.upsertAppDownstreamVersionStatements(appID, d.ClusterID, sequence,
			kotsKinds.Installation.Spec.VersionLabel, downstreamStatus,
			source, diffSummary, diffSummaryError, commitURL, commitURL != "", skipPreflights)
		if err != nil {
			return nil, errors.Wrap(err, "failed to construct app downstream version statements")
		}
		statements = append(statements, downstreamVersionStatements...)
	}

	return statements, nil
}

func (s *KOTSStore) createAppVersionRecordStatements(appID string, appName string, appIcon string, kotsKinds *kotsutil.KotsKinds, brandingArchive []byte) ([]gorqlite.ParameterizedStatement, int64, error) {
	newSequence, err := s.GetNextAppSequence(appID)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to get next sequence number")
	}

	appVersionRecordStatements, err := s.upsertAppVersionRecordStatements(appID, newSequence, appName, appIcon, kotsKinds, brandingArchive)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to construct app version record statements")
	}

	return appVersionRecordStatements, newSequence, nil
}

func (s *KOTSStore) upsertAppVersionRecordStatements(appID string, sequence int64, appName string, appIcon string, kotsKinds *kotsutil.KotsKinds, brandingArchive []byte) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	// we marshal these here because it's a decision of the store to cache them in the app version table
	// not all stores will do this
	supportBundleSpec, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Collector")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal support bundle spec")
	}
	analyzersSpec, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Analyzer")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal analyzer spec")
	}
	preflightSpec, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Preflight")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal preflight spec")
	}

	appSpec, err := kotsKinds.Marshal("app.k8s.io", "v1beta1", "Application")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal app spec")
	}
	kotsAppSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Application")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal kots app spec")
	}
	kotsInstallationSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Installation")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal kots installation spec")
	}

	backupSpec, err := kotsKinds.Marshal("velero.io", "v1", "Backup")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal backup spec")
	}
	restoreSpec, err := kotsKinds.Marshal("velero.io", "v1", "Restore")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal restore spec")
	}
	identitySpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Identity")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal identity spec")
	}

	licenseSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "License")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal license spec")
	}
	configSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Config")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal config spec")
	}
	configValuesSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal configvalues spec")
	}

	embeddedClusterConfig, err := kotsKinds.Marshal("embeddedcluster.replicated.com", "v1beta1", "Config")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal configvalues spec")
	}

	var releasedAt *int64
	if kotsKinds.Installation.Spec.ReleasedAt != nil {
		t := kotsKinds.Installation.Spec.ReleasedAt.Time.Unix()
		releasedAt = &t
	}

	query := `insert into app_version (app_id, sequence, created_at, version_label, is_required, release_notes, update_cursor, channel_id, channel_name, upstream_released_at, encryption_key,
		supportbundle_spec, analyzer_spec, preflight_spec, app_spec, kots_app_spec, kots_installation_spec, kots_license, config_spec, config_values, backup_spec, restore_spec, identity_spec, branding_archive, embeddedcluster_config)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(app_id, sequence) DO UPDATE SET
		created_at = EXCLUDED.created_at,
		version_label = EXCLUDED.version_label,
		is_required = EXCLUDED.is_required,
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
		restore_spec = EXCLUDED.restore_spec,
		identity_spec = EXCLUDED.identity_spec,
		branding_archive = EXCLUDED.branding_archive,
		embeddedcluster_config = EXCLUDED.embeddedcluster_config`

	statements = append(statements, gorqlite.ParameterizedStatement{
		Query: query,
		Arguments: []interface{}{
			appID,
			sequence,
			time.Now().Unix(),
			kotsKinds.Installation.Spec.VersionLabel,
			kotsKinds.Installation.Spec.IsRequired,
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
			restoreSpec,
			identitySpec,
			base64.StdEncoding.EncodeToString(brandingArchive),
			embeddedClusterConfig,
		},
	})

	// an old version could be downloaded at a later point, pick higher sequence
	statements = append(statements, gorqlite.ParameterizedStatement{
		Query:     "update app set current_sequence = ifnull(max(current_sequence, ?), 0), name = ?, icon_uri = ? where id = ?",
		Arguments: []interface{}{sequence, appName, appIcon, appID},
	})

	return statements, nil
}

func determineDownstreamVersionStatus(s store.Store, a *apptypes.App, sequence int64, kotsKinds *kotsutil.KotsKinds, isInstall bool, isAutomated bool, skipPreflights bool) (types.DownstreamVersionStatus, error) {
	// Check if we need to show cluster management
	if util.IsEmbeddedCluster() && isInstall && !isAutomated && kotsKinds.IsMultiNodeEnabled() {
		return types.VersionPendingClusterManagement, nil
	}

	// Check if we need to configure the app
	if kotsKinds.IsConfigurable() {
		registrySettings, err := s.GetRegistryDetailsForApp(a.ID)
		if err != nil {
			return types.VersionUnknown, errors.Wrap(err, "failed to get app registry info")
		}
		needsConfig, err := kotsadmconfig.NeedsConfiguration(a.Slug, sequence, a.IsAirgap, kotsKinds, registrySettings)
		if err != nil {
			return types.VersionUnknown, errors.Wrap(err, "failed to check if version needs configuration")
		}
		if needsConfig {
			return types.VersionPendingConfig, nil
		}
		if isInstall && !isAutomated {
			// initial installs from the UI should show the config page even if all required items are already set and have values
			return types.VersionPendingConfig, nil
		}
	}

	// Check if we need to run preflights
	if kotsKinds.HasPreflights() {
		if !skipPreflights {
			return types.VersionPendingPreflight, nil
		}
		strict, err := kotsKinds.HasStrictPreflights()
		if err != nil {
			return types.VersionUnknown, errors.Wrap(err, "failed to check strict preflights from spec")
		}
		if strict {
			logger.Warnf("preflights will not be skipped, strict preflights are set to true")
			return types.VersionPendingPreflight, nil
		}
	}

	return types.VersionPending, nil
}

func (s *KOTSStore) upsertAppDownstreamVersionStatements(appID string, clusterID string, sequence int64, versionLabel string, status types.DownstreamVersionStatus, source string, diffSummary string, diffSummaryError string, commitURL string, gitDeployable bool, preflightsSkipped bool) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `insert into app_downstream_version (app_id, cluster_id, sequence, parent_sequence, created_at, version_label, status, source, diff_summary, diff_summary_error, git_commit_url, git_deployable, preflight_skipped)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

	statements = append(statements, gorqlite.ParameterizedStatement{
		Query: query,
		Arguments: []interface{}{
			appID,
			clusterID,
			sequence,
			sequence,
			time.Now().Unix(),
			versionLabel,
			status,
			source,
			diffSummary,
			diffSummaryError,
			commitURL,
			gitDeployable,
			preflightsSkipped,
		},
	})

	return statements, nil
}

func (s *KOTSStore) GetAppVersion(appID string, sequence int64) (*versiontypes.AppVersion, error) {
	db := persistence.MustGetDBSession()
	query := `select app_id, sequence, update_cursor, channel_id, version_label, created_at, status, applied_at, kots_installation_spec, kots_app_spec, kots_license, embeddedcluster_config from app_version where app_id = ? and sequence = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, sequence},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return nil, ErrNotFound
	}

	v, err := s.appVersionFromRow(rows)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app version from row")
	}

	return v, nil
}

// GetLatestAppSequence returns the sequence of the latest app version.
// This function handles both semantic and non-semantic versions.
// If downloadedOnly param is set to true, the sequence of the latest downloaded app version will be returned.
func (s *KOTSStore) GetLatestAppSequence(appID string, downloadedOnly bool) (int64, error) {
	versions, err := s.FindDownstreamVersions(appID, downloadedOnly)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get latest downstream version")
	}
	if len(versions.AllVersions) == 0 {
		return 0, errors.New("no versions found for app")
	}
	return versions.AllVersions[0].ParentSequence, nil
}

func (s *KOTSStore) UpdateNextAppVersionDiffSummary(appID string, baseSequence int64) error {
	a, err := s.GetApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	appVersions, err := s.FindDownstreamVersions(a.ID, true)
	if err != nil {
		return errors.Wrapf(err, "failed to find app downstream versions for app %s", appID)
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

	baseArchiveDir, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create base temp dir")
	}
	defer os.RemoveAll(baseArchiveDir)

	if err := s.GetAppVersionArchive(appID, baseSequence, baseArchiveDir); err != nil {
		return errors.Wrap(err, "failed to get base archive dir")
	}

	nextArchiveDir, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create next temp dir")
	}
	defer os.RemoveAll(nextArchiveDir)

	if err := s.GetAppVersionArchive(appID, nextSequence, nextArchiveDir); err != nil {
		return errors.Wrap(err, "failed to get next archive dir")
	}

	diffSummary, diffSummaryError := "", ""

	diff, err := apparchive.DiffAppVersionsForDownstream(d.Name, nextArchiveDir, baseArchiveDir, binaries.GetKustomizeBinPath())
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
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     `update app_downstream_version set diff_summary = ?, diff_summary_error = ? where app_id = ? AND sequence = ?`,
		Arguments: []interface{}{diffSummary, diffSummaryError, appID, nextSequence},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) GetNextAppSequence(appID string) (int64, error) {
	db := persistence.MustGetDBSession()

	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     `select max(sequence) from app_version where app_id = ?`,
		Arguments: []interface{}{appID},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return 0, ErrNotFound
	}

	var maxSequence gorqlite.NullInt64
	if err := rows.Scan(&maxSequence); err != nil {
		return 0, errors.Wrap(err, "failed to find current max sequence in row")
	}

	newSequence := int64(0)
	if maxSequence.Valid {
		newSequence = maxSequence.Int64 + 1
	}

	return newSequence, nil
}

func (s *KOTSStore) GetCurrentUpdateCursor(appID string, channelID string) (string, error) {
	db := persistence.MustGetDBSession()
	query := `SELECT update_cursor FROM app_version WHERE app_id = ? AND channel_id = ? AND sequence IN (
		SELECT MAX(sequence) FROM app_version WHERE app_id = ? AND channel_id = ?
	) ORDER BY sequence DESC LIMIT 1`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, channelID, appID, channelID},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return "", nil
	}

	var updateCursor gorqlite.NullString
	if err := rows.Scan(&updateCursor); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return updateCursor.String, nil
}

func (s *KOTSStore) HasStrictPreflights(appID string, sequence int64) (bool, error) {
	var preflightSpecStr gorqlite.NullString
	db := persistence.MustGetDBSession()
	query := `SELECT preflight_spec FROM app_version WHERE app_id = ? AND sequence = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, sequence},
	})
	if err != nil {
		return false, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return false, ErrNotFound
	}

	if err := rows.Scan(&preflightSpecStr); err != nil {
		return false, errors.Wrap(err, "failed to scan")
	}

	return s.hasStrictPreflights(preflightSpecStr)
}

func (s *KOTSStore) appVersionFromRow(row gorqlite.QueryResult) (*versiontypes.AppVersion, error) {
	v := &versiontypes.AppVersion{}

	var status gorqlite.NullString
	var deployedAt gorqlite.NullTime
	var createdAt gorqlite.NullTime
	var installationSpec gorqlite.NullString
	var kotsAppSpec gorqlite.NullString
	var licenseSpec gorqlite.NullString
	var updateCursor gorqlite.NullString
	var channelID gorqlite.NullString
	var versionLabel gorqlite.NullString
	var embeddedClusterConfig gorqlite.NullString

	if err := row.Scan(&v.AppID, &v.Sequence, &updateCursor, &channelID, &versionLabel, &createdAt, &status, &createdAt, &installationSpec, &kotsAppSpec, &licenseSpec, &embeddedClusterConfig); err != nil {
		return nil, errors.Wrap(err, "failed to scan")
	}

	v.KOTSKinds = &kotsutil.KotsKinds{}

	v.UpdateCursor = updateCursor.String
	v.ChannelID = channelID.String
	v.VersionLabel = versionLabel.String

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

	if embeddedClusterConfig.Valid && embeddedClusterConfig.String != "" {
		config, err := kotsutil.LoadEmbeddedClusterConfigFromBytes([]byte(embeddedClusterConfig.String))
		if err != nil {
			return nil, errors.Wrap(err, "failed to read embedded cluster config")
		}
		if config != nil {
			v.KOTSKinds.EmbeddedClusterConfig = config
		}
	}

	v.CreatedOn = createdAt.Time
	if deployedAt.Valid {
		v.DeployedAt = &deployedAt.Time
	}

	if sv, err := semver.ParseTolerant(v.VersionLabel); err == nil {
		v.Semver = &sv
	}

	if c, err := cursor.NewCursor(v.UpdateCursor); err == nil {
		v.Cursor = &c
	}

	v.Status = status.String

	return v, nil
}
