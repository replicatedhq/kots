package kotsstore

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/cursor"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/tasks"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/rqlite/gorqlite"
)

func (s *KOTSStore) GetCurrentDownstreamSequence(appID string, clusterID string) (int64, error) {
	db := persistence.MustGetDBSession()
	query := `select current_sequence from app_downstream where app_id = ? and cluster_id = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, clusterID},
	})
	if err != nil {
		return -1, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return -1, ErrNotFound
	}

	var currentSequence gorqlite.NullInt64
	if err := rows.Scan(&currentSequence); err != nil {
		return -1, errors.Wrap(err, "failed to scan")
	}

	if !currentSequence.Valid {
		return -1, nil
	}

	return currentSequence.Int64, nil
}

func (s *KOTSStore) GetCurrentParentSequence(appID string, clusterID string) (int64, error) {
	currentSequence, err := s.GetCurrentDownstreamSequence(appID, clusterID)
	if err != nil {
		return -1, errors.Wrap(err, "failed to get current parent sequence")
	}
	if currentSequence == -1 {
		return -1, nil
	}

	db := persistence.MustGetDBSession()
	query := `select parent_sequence from app_downstream_version where app_id = ? and cluster_id = ? and sequence = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, clusterID, currentSequence},
	})
	if err != nil {
		return -1, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return -1, ErrNotFound
	}

	var parentSequence gorqlite.NullInt64
	if err := rows.Scan(&parentSequence); err != nil {
		return -1, errors.Wrap(err, "failed to scan")
	}

	if !parentSequence.Valid {
		return -1, nil
	}

	return parentSequence.Int64, nil
}

func (s *KOTSStore) GetParentSequenceForSequence(appID string, clusterID string, sequence int64) (int64, error) {
	db := persistence.MustGetDBSession()
	query := `select parent_sequence from app_downstream_version where app_id = ? and cluster_id = ? and sequence = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, clusterID, sequence},
	})
	if err != nil {
		return -1, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return -1, ErrNotFound
	}

	var parentSequence gorqlite.NullInt64
	if err := rows.Scan(&parentSequence); err != nil {
		return -1, errors.Wrap(err, "failed to scan")
	}

	if !parentSequence.Valid {
		return -1, nil
	}

	return parentSequence.Int64, nil
}

func (s *KOTSStore) GetPreviouslyDeployedSequence(appID string, clusterID string) (int64, error) {
	db := persistence.MustGetDBSession()
	query := `select sequence from app_downstream_version where app_id = ? and cluster_id = ? and applied_at is not null order by applied_at desc limit 2`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, clusterID},
	})
	if err != nil {
		return -1, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	for rowNumber := 1; rows.Next(); rowNumber++ {
		if rowNumber != 2 {
			continue
		}
		var sequence int64
		if err := rows.Scan(&sequence); err != nil {
			return -1, errors.Wrap(err, "failed to scan")
		}
		return sequence, nil
	}

	return -1, nil
}

func (s *KOTSStore) MarkAsCurrentDownstreamVersion(appID string, sequence int64) error {
	db := persistence.MustGetDBSession()
	statements := []gorqlite.ParameterizedStatement{}

	statements = append(statements, gorqlite.ParameterizedStatement{
		Query:     `update app_downstream set current_sequence = ? where app_id = ?`,
		Arguments: []interface{}{sequence, appID},
	})

	statements = append(statements, gorqlite.ParameterizedStatement{
		Query:     `update app_downstream_version set applied_at = ? where sequence = ? and app_id = ?`,
		Arguments: []interface{}{time.Now().Unix(), sequence, appID},
	})

	if wrs, err := db.WriteParameterized(statements); err != nil {
		wrErrs := []error{}
		for _, wr := range wrs {
			wrErrs = append(wrErrs, wr.Err)
		}
		return fmt.Errorf("failed to write: %v: %v", err, wrErrs)
	}

	return nil
}

// SetDownstreamVersionStatus updates the status and status info for the downstream version with the given sequence and app id
func (s *KOTSStore) SetDownstreamVersionStatus(appID string, sequence int64, status types.DownstreamVersionStatus, statusInfo string) error {
	db := persistence.MustGetDBSession()
	query := `update app_downstream_version set status = ?, status_info = ? where app_id = ? and sequence = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{status, statusInfo, appID, sequence},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

// GetDownstreamVersionStatus gets the status for the downstream version with the given sequence and app id
func (s *KOTSStore) GetDownstreamVersionStatus(appID string, sequence int64) (types.DownstreamVersionStatus, error) {
	db := persistence.MustGetDBSession()
	query := `select status from app_downstream_version where app_id = ? and sequence = ?`
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

	var status gorqlite.NullString
	if err := rows.Scan(&status); err != nil {
		return "", errors.Wrap(err, "failed to get downstream version")
	}

	return types.DownstreamVersionStatus(status.String), nil
}

// GetDownstreamVersionSource gets the source for the downstream version with the given sequence and app id
func (s *KOTSStore) GetDownstreamVersionSource(appID string, sequence int64) (string, error) {
	db := persistence.MustGetDBSession()
	query := `select source from app_downstream_version where app_id = ? and sequence = ?`
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

	var source gorqlite.NullString
	if err := rows.Scan(&source); err != nil {
		return "", errors.Wrap(err, "failed to get downstream version")
	}

	return source.String, nil
}

func (s *KOTSStore) GetIgnoreRBACErrors(appID string, sequence int64) (bool, error) {
	db := persistence.MustGetDBSession()
	query := `SELECT preflight_ignore_permissions FROM app_downstream_version WHERE app_id = ? and sequence = ? LIMIT 1`
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

	var shouldIgnore gorqlite.NullBool
	if err := rows.Scan(&shouldIgnore); err != nil {
		return false, errors.Wrap(err, "failed to select downstream")
	}

	if !shouldIgnore.Valid {
		return false, nil
	}

	return shouldIgnore.Bool, nil
}

func (s *KOTSStore) GetCurrentDownstreamVersion(appID string, clusterID string) (*downstreamtypes.DownstreamVersion, error) {
	currentSequence, err := s.GetCurrentDownstreamSequence(appID, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current sequence")
	}
	if currentSequence == -1 {
		return nil, nil
	}

	db := persistence.MustGetDBSession()
	query := `SELECT
	adv.created_at,
	adv.status,
	adv.sequence,
	adv.parent_sequence,
	adv.applied_at,
	adv.source,
	adv.preflight_skipped,
	adv.git_commit_url,
	adv.git_deployable,
	ado.is_error,
	av.upstream_released_at,
	av.version_label,
	av.channel_id,
	av.update_cursor,
	av.is_required
 FROM
	 app_downstream_version AS adv
 LEFT JOIN
	 app_version AS av
 ON
	 adv.app_id = av.app_id AND adv.parent_sequence = av.sequence
 LEFT JOIN
	 app_downstream_output AS ado
 ON
	 adv.app_id = ado.app_id AND adv.cluster_id = ado.cluster_id AND adv.sequence = ado.downstream_sequence
 WHERE
	 adv.app_id = ? AND
	 adv.cluster_id = ? AND
	 adv.sequence = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, clusterID, currentSequence},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return nil, ErrNotFound
	}

	v, err := s.downstreamVersionFromRow(appID, rows)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get version from row")
	}

	// checking if a version is deployable requires querying all versions
	// save some time here and don't check that because current version is always re-deployable
	if err := s.AddDownstreamVersionDetails(appID, clusterID, v, false); err != nil {
		return nil, errors.Wrap(err, "failed to add details")
	}
	v.IsDeployable = true

	return v, nil
}

func (s *KOTSStore) GetStatusForVersion(appID string, clusterID string, sequence int64) (types.DownstreamVersionStatus, error) {
	db := persistence.MustGetDBSession()
	query := `SELECT
	adv.status, ado.is_error
	FROM
		app_downstream_version AS adv
	LEFT JOIN
		app_downstream_output AS ado
	ON
		adv.app_id = ado.app_id AND adv.cluster_id = ado.cluster_id AND adv.sequence = ado.downstream_sequence
 WHERE
 	adv.app_id = ? AND adv.cluster_id = ? AND adv.sequence = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, clusterID, sequence},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return "", ErrNotFound
	}

	var status gorqlite.NullString
	var hasError gorqlite.NullBool
	if err := rows.Scan(&status, &hasError); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return types.DownstreamVersionStatus(status.String), nil
}

func (s *KOTSStore) GetDownstreamVersions(appID string, clusterID string, downloadedOnly bool) (*downstreamtypes.DownstreamVersions, error) {
	currentVersion, err := s.GetCurrentDownstreamVersion(appID, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current version")
	}

	db := persistence.MustGetDBSession()
	query := `SELECT
	adv.created_at,
	adv.status,
	adv.sequence,
	adv.parent_sequence,
	adv.applied_at,
	adv.source,
	adv.preflight_skipped,
	adv.git_commit_url,
	adv.git_deployable,
	ado.is_error,
	av.upstream_released_at,
	av.version_label,
	av.channel_id,
	av.update_cursor,
	av.is_required
 FROM
	 app_downstream_version AS adv
 LEFT JOIN
	 app_version AS av
 ON
	 adv.app_id = av.app_id AND adv.parent_sequence = av.sequence
 LEFT JOIN
	 app_downstream_output AS ado
 ON
	 adv.app_id = ado.app_id AND adv.cluster_id = ado.cluster_id AND adv.sequence = ado.downstream_sequence
 WHERE
	 adv.app_id = ? AND
	 adv.cluster_id = ?`

	if downloadedOnly {
		query += fmt.Sprintf(` AND adv.status != '%s'`, types.VersionPendingDownload)
	}

	query += ` ORDER BY adv.sequence DESC`

	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, clusterID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	result := &downstreamtypes.DownstreamVersions{
		CurrentVersion: currentVersion,
		AllVersions:    []*downstreamtypes.DownstreamVersion{},
	}
	for rows.Next() {
		v, err := s.downstreamVersionFromRow(appID, rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get version from row")
		}
		if v != nil {
			result.AllVersions = append(result.AllVersions, v)
		}
	}

	license, err := s.GetLatestLicenseForApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app license")
	}

	downstreamtypes.SortDownstreamVersions(result.AllVersions, license.Spec.IsSemverRequired)

	// retrieve additional details about the latest downloaded version,
	// since it's used for detecting things like if a certain feature is enabled or not.
	for _, v := range result.AllVersions {
		if v.Status == types.VersionPendingDownload {
			continue
		}
		// checking if a version is deployable requires getting all versions again.
		// check if latest version is deployable separately to avoid cycle dependencies between functions.
		if err := s.AddDownstreamVersionDetails(appID, clusterID, v, false); err != nil {
			return nil, errors.Wrap(err, "failed to add details to latest downloaded version")
		}
		v.IsDeployable, v.NonDeployableCause, err = isAppVersionDeployable(s, appID, v, result, license.Spec.IsSemverRequired)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check if version %s is deployable", v.VersionLabel)
		}
	}

	if currentVersion == nil {
		result.PendingVersions = result.AllVersions
		result.PastVersions = []*downstreamtypes.DownstreamVersion{}
		return result, nil
	}

	for i, v := range result.AllVersions {
		if v.Sequence == currentVersion.Sequence {
			result.PendingVersions = result.AllVersions[:i]
			result.PastVersions = result.AllVersions[i+1:]
			break
		}
	}

	return result, nil
}

func (s *KOTSStore) FindDownstreamVersions(appID string, downloadedOnly bool) (*downstreamtypes.DownstreamVersions, error) {
	downstreams, err := s.ListDownstreamsForApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app downstreams")
	}
	if len(downstreams) == 0 {
		return nil, errors.New("app has no downstreams")
	}

	for _, d := range downstreams {
		clusterID := d.ClusterID
		downstreamVersions, err := s.GetDownstreamVersions(appID, clusterID, downloadedOnly)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get downstream versions for cluster %s", clusterID)
		}
		if len(downstreamVersions.AllVersions) > 0 {
			return downstreamVersions, nil
		}
	}

	return nil, errors.New("app has no versions")
}

func (s *KOTSStore) GetDownstreamVersionHistory(appID string, clusterID string, currentPage int, pageSize int, pinLatest bool, pinLatestDeployable bool) (*downstreamtypes.DownstreamVersionHistory, error) {
	history := &downstreamtypes.DownstreamVersionHistory{}

	versions, err := s.GetDownstreamVersions(appID, clusterID, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get downstream versions without details")
	}
	history.TotalCount = len(versions.AllVersions)

	desiredVersions := []*downstreamtypes.DownstreamVersion{}

	if pinLatest {
		if len(versions.AllVersions) > 0 {
			desiredVersions = append(desiredVersions, versions.AllVersions[0])
		}
	}

	if pinLatestDeployable {
		latestDeployableVersion, numOfSkippedVersions, numOfRemainingVersions, err := s.getLatestDeployableDownstreamVersion(appID, clusterID, versions)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get latest deployable downstream version")
		}
		if latestDeployableVersion != nil {
			desiredVersions = append(desiredVersions, latestDeployableVersion)
		}
		history.NumOfSkippedVersions = numOfSkippedVersions
		history.NumOfRemainingVersions = numOfRemainingVersions
	}

	startIndex := currentPage * pageSize
	endIndex := currentPage*pageSize + pageSize
	for i, v := range versions.AllVersions {
		if currentPage != -1 && i < startIndex {
			continue
		}
		if pageSize != -1 && i >= endIndex {
			break
		}
		desiredVersions = append(desiredVersions, v)
	}

	if err := s.AddDownstreamVersionsDetails(appID, clusterID, desiredVersions, true); err != nil {
		return nil, errors.Wrap(err, "failed to add details for desired versions")
	}
	history.VersionHistory = desiredVersions

	return history, nil
}

func (s *KOTSStore) AddDownstreamVersionDetails(appID string, clusterID string, version *downstreamtypes.DownstreamVersion, checkIfDeployable bool) error {
	return s.AddDownstreamVersionsDetails(appID, clusterID, []*downstreamtypes.DownstreamVersion{version}, checkIfDeployable)
}

func (s *KOTSStore) AddDownstreamVersionsDetails(appID string, clusterID string, versions []*downstreamtypes.DownstreamVersion, checkIfDeployable bool) error {
	sequencesToQuery := []string{}
	for _, v := range versions {
		if v == nil {
			continue
		}
		sequencesToQuery = append(sequencesToQuery, fmt.Sprintf("%d", v.Sequence))
	}

	if len(sequencesToQuery) == 0 {
		return nil
	}

	db := persistence.MustGetDBSession()
	query := fmt.Sprintf(`SELECT
	adv.sequence,
	adv.diff_summary,
	adv.diff_summary_error,
	adv.preflight_result,
	adv.preflight_result_created_at,
	av.kots_installation_spec,
	av.kots_app_spec,
	av.preflight_spec,
	av.config_spec
 FROM
	 app_downstream_version AS adv
 LEFT JOIN
	 app_version AS av
 ON
	 adv.app_id = av.app_id AND adv.parent_sequence = av.sequence
 WHERE
	 adv.app_id = ? AND
	 adv.cluster_id = ? AND
	 adv.sequence IN (%s)
 ORDER BY
 	 adv.sequence DESC`, strings.Join(sequencesToQuery, ","))

	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, clusterID},
	})
	if err != nil {
		return fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	for rows.Next() {
		var sequence int64
		var diffSummary gorqlite.NullString
		var diffSummaryError gorqlite.NullString
		var preflightResult gorqlite.NullString
		var preflightResultCreatedAt gorqlite.NullTime
		var kotsInstallationSpecStr gorqlite.NullString
		var kotsAppSpecStr gorqlite.NullString
		var preflightSpecStr gorqlite.NullString
		var configSpecStr gorqlite.NullString

		if err := rows.Scan(
			&sequence,
			&diffSummary,
			&diffSummaryError,
			&preflightResult,
			&preflightResultCreatedAt,
			&kotsInstallationSpecStr,
			&kotsAppSpecStr,
			&preflightSpecStr,
			&configSpecStr,
		); err != nil {
			return errors.Wrap(err, "failed to scan")
		}

		// find the version
		var version *downstreamtypes.DownstreamVersion
		for _, v := range versions {
			if v.Sequence == sequence {
				version = v
				break
			}
		}

		version.DiffSummary = diffSummary.String
		version.DiffSummaryError = diffSummaryError.String

		version.PreflightResult = preflightResult.String
		if preflightResultCreatedAt.Valid {
			version.PreflightResultCreatedAt = &preflightResultCreatedAt.Time
		}

		releaseNotes, err := getReleaseNotes(appID, version.ParentSequence)
		if err != nil {
			return errors.Wrap(err, "failed to get release notes")
		}
		version.ReleaseNotes = releaseNotes

		if version.Status == types.VersionPendingDownload {
			downloadTaskID := fmt.Sprintf("update-download.%d", version.Sequence)
			downloadStatus, downloadStatusMessage, err := tasks.GetTaskStatus(downloadTaskID)
			if err != nil {
				// don't fail on this
				logger.Error(errors.Wrap(err, fmt.Sprintf("failed to get %s task status", downloadTaskID)))
			}
			version.DownloadStatus = downstreamtypes.DownloadStatus{
				Message: downloadStatusMessage,
				Status:  downloadStatus,
			}
		}

		version.KOTSKinds = &kotsutil.KotsKinds{}

		if kotsInstallationSpecStr.String != "" {
			installation, err := kotsutil.LoadInstallationFromContents([]byte(kotsInstallationSpecStr.String))
			if err != nil {
				return errors.Wrap(err, "failed to load installation spec")
			}
			version.KOTSKinds.Installation = *installation
			version.YamlErrors = version.KOTSKinds.Installation.Spec.YAMLErrors
		}

		if kotsAppSpecStr.String != "" {
			app, err := kotsutil.LoadKotsAppFromContents([]byte(kotsAppSpecStr.String))
			if err != nil {
				return errors.Wrap(err, "failed to load installation spec")
			}
			version.KOTSKinds.KotsApplication = *app
			version.AppTitle = app.Spec.Title
			version.AppIconURI = app.Spec.Icon
		}
		version.NeedsKotsUpgrade = needsKotsUpgrade(&version.KOTSKinds.KotsApplication)

		p, err := s.hasFailingStrictPreflights(preflightSpecStr, preflightResult)
		if err != nil {
			return errors.Wrap(err, "failed to get strict preflight results")
		}
		version.HasFailingStrictPreflights = p

		if configSpecStr.String != "" {
			config, err := kotsutil.LoadConfigFromBytes([]byte(configSpecStr.String))
			if err != nil {
				return errors.Wrap(err, "failed to load config from spec")
			}
			if len(config.Spec.Groups) > 0 {
				version.HasConfig = true
			}
		}
	}

	if checkIfDeployable {
		allVersions, err := s.GetDownstreamVersions(appID, clusterID, false)
		if err != nil {
			return errors.Wrapf(err, "failed to get downstream versions without details")
		}
		license, err := s.GetLatestLicenseForApp(appID)
		if err != nil {
			return errors.Wrap(err, "failed to get app license")
		}

		for _, v := range versions {
			v.IsDeployable, v.NonDeployableCause, err = isAppVersionDeployable(s, appID, v, allVersions, license.Spec.IsSemverRequired)
			if err != nil {
				return errors.Wrapf(err, "failed to check if version %s is deployable", v.VersionLabel)
			}
		}
	}

	return nil
}

func (s *KOTSStore) GetLatestDeployableDownstreamVersion(appID string, clusterID string) (latestDeployableVersion *downstreamtypes.DownstreamVersion, numOfSkippedVersions int, numOfRemainingVersions int, finalError error) {
	versions, err := s.GetDownstreamVersions(appID, clusterID, false)
	if err != nil {
		finalError = errors.Wrap(err, "failed to get app downstream versions")
		return
	}

	return s.getLatestDeployableDownstreamVersion(appID, clusterID, versions)
}

func (s *KOTSStore) getLatestDeployableDownstreamVersion(appID string, clusterID string, versions *downstreamtypes.DownstreamVersions) (latestDeployableVersion *downstreamtypes.DownstreamVersion, numOfSkippedVersions int, numOfRemainingVersions int, finalError error) {
	defer func() {
		if latestDeployableVersion != nil {
			if err := s.AddDownstreamVersionDetails(appID, clusterID, latestDeployableVersion, true); err != nil {
				finalError = errors.Wrap(err, "failed to add details")
				return
			}
		}
	}()

	if len(versions.AllVersions) == 0 {
		finalError = errors.New("no versions found for app")
		return
	}

	if versions.CurrentVersion == nil {
		// no version has been deployed yet, next app version is the latest version
		latestDeployableVersion = versions.AllVersions[0]
		return
	}

	if len(versions.PendingVersions) == 0 {
		// latest version is already deployed, there's no next app version
		return
	}

	// find required versions
	requiredVersions := []*downstreamtypes.DownstreamVersion{}
	for _, v := range versions.PendingVersions {
		if v.IsRequired {
			requiredVersions = append(requiredVersions, v)
		}
	}

	if len(requiredVersions) > 0 {
		// next app version is the earliest pending required version
		latestDeployableVersion = requiredVersions[len(requiredVersions)-1]
	} else {
		// next app version is the latest pending version
		latestDeployableVersion = versions.PendingVersions[0]
	}

	latestDeployableVersionIndex := -1
	for i, v := range versions.PendingVersions {
		if v.Sequence == latestDeployableVersion.Sequence {
			latestDeployableVersionIndex = i
			break
		}
	}

	for i := range versions.PendingVersions {
		if i < latestDeployableVersionIndex {
			numOfRemainingVersions++
		}
		if i > latestDeployableVersionIndex {
			numOfSkippedVersions++
		}
	}

	return
}

func (s *KOTSStore) downstreamVersionFromRow(appID string, row gorqlite.QueryResult) (*downstreamtypes.DownstreamVersion, error) {
	v := &downstreamtypes.DownstreamVersion{}

	var createdOn gorqlite.NullTime
	var versionLabel gorqlite.NullString
	var channelID gorqlite.NullString
	var updateCursor gorqlite.NullString
	var status gorqlite.NullString
	var parentSequence gorqlite.NullInt64
	var deployedAt gorqlite.NullTime
	var source gorqlite.NullString
	var preflightSkipped gorqlite.NullBool
	var commitURL gorqlite.NullString
	var gitDeployable gorqlite.NullBool
	var hasError gorqlite.NullBool
	var upstreamReleasedAt gorqlite.NullTime

	if err := row.Scan(
		&createdOn,
		&status,
		&v.Sequence,
		&parentSequence,
		&deployedAt,
		&source,
		&preflightSkipped,
		&commitURL,
		&gitDeployable,
		&hasError,
		&upstreamReleasedAt,
		&versionLabel,
		&channelID,
		&updateCursor,
		&v.IsRequired,
	); err != nil {
		return nil, errors.Wrap(err, "failed to scan")
	}

	if createdOn.Valid {
		v.CreatedOn = &createdOn.Time
	}

	v.VersionLabel = versionLabel.String
	sv, err := semver.ParseTolerant(v.VersionLabel)
	if err == nil {
		v.Semver = &sv
	}

	v.UpdateCursor = updateCursor.String
	if c, err := cursor.NewCursor(v.UpdateCursor); err == nil {
		v.Cursor = &c
	}

	v.ChannelID = channelID.String

	v.Status = types.DownstreamVersionStatus(status.String)
	v.ParentSequence = parentSequence.Int64

	if deployedAt.Valid {
		v.DeployedAt = &deployedAt.Time
	}
	v.Source = source.String
	v.PreflightSkipped = preflightSkipped.Bool
	v.CommitURL = commitURL.String
	v.GitDeployable = gitDeployable.Bool

	if upstreamReleasedAt.Valid {
		v.UpstreamReleasedAt = &upstreamReleasedAt.Time
	}

	return v, nil
}

func (s *KOTSStore) IsAppVersionDeployable(appID string, sequence int64) (bool, string, error) {
	downstreams, err := s.ListDownstreamsForApp(appID)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get app downstreams")
	}
	if len(downstreams) == 0 {
		return false, "", errors.New("app has no downstreams")
	}
	clusterID := downstreams[0].ClusterID

	versions, err := s.GetDownstreamVersions(appID, clusterID, false)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get downstream versions")
	}
	for _, v := range versions.AllVersions {
		if v.Sequence == sequence {
			if err := s.AddDownstreamVersionDetails(appID, clusterID, v, true); err != nil {
				return false, "", errors.Wrap(err, "failed to add details")
			}
			return v.IsDeployable, v.NonDeployableCause, nil
		}
	}

	return false, "", errors.Errorf("version %d not found", sequence)
}

func isSameUpstreamRelease(v1 *downstreamtypes.DownstreamVersion, v2 *downstreamtypes.DownstreamVersion, isSemverRequired bool) bool {
	if v1.ChannelID == v2.ChannelID && v1.UpdateCursor == v2.UpdateCursor {
		return true
	}
	if !isSemverRequired {
		return false
	}
	if v1.Semver == nil || v2.Semver == nil {
		return false
	}
	return v1.Semver.EQ(*v2.Semver)
}

func isAppVersionDeployable(s store.Store, appID string, version *downstreamtypes.DownstreamVersion, appVersions *downstreamtypes.DownstreamVersions, isSemverRequired bool) (bool, string, error) {
	if version.HasFailingStrictPreflights {
		return false, "Deployment is disabled as a strict analyzer in this version's preflight checks has failed or has not been run.", nil
	}

	if version.Status == types.VersionPendingDownload {
		return false, "Version is pending download.", nil
	}

	if version.Status == types.VersionPendingConfig {
		return false, "Version is pending configuration.", nil
	}

	if appVersions.CurrentVersion == nil {
		// no version has been deployed yet, treat as an initial install where any version can be deployed at first.
		return true, "", nil
	}

	if version.Sequence == appVersions.CurrentVersion.Sequence {
		// version is currently deployed, so previous required versions should've already been deployed.
		// also, we shouldn't block re-deploying if a previous release is edited later by the vendor to be required.
		return true, "", nil
	}

	// rollback support is determined across all versions from all channels
	// versions below the current veresion in the list are considered past versions
	versionIndex := -1
	for i, v := range appVersions.AllVersions {
		if v.Sequence == version.Sequence {
			versionIndex = i
			break
		}
	}
	deployedVersionIndex := -1
	for i, v := range appVersions.AllVersions {
		if v.Sequence == appVersions.CurrentVersion.Sequence {
			deployedVersionIndex = i
			break
		}
	}

	// This is a past version
	if versionIndex > deployedVersionIndex {
		// Rollback support is based off of the latest downloaded version so that a vendor can
		// toggle on support without requiring the end user to deploy a new version.
		for _, v := range appVersions.AllVersions {
			if v.Status == types.VersionPendingDownload {
				continue
			}
			if v.KOTSKinds == nil || !v.KOTSKinds.KotsApplication.Spec.AllowRollback {
				return false, "Rollback is not supported.", nil
			}
			break
		}

		if util.IsEmbeddedCluster() {
			// Compare the embedded cluster config of the version specified to the currently
			// deployed version to check if it has changed. If it has, then we do not allow
			// rollbacks.
			changed, err := didECClusterConfigChange(s, appID, version, appVersions.CurrentVersion)
			if err != nil {
				return false, "", errors.Wrapf(err, "failed to check if embedded cluster config changed for version %d", version.Sequence)
			}
			if changed {
				return false, "Rollback is not supported, cluster configuration has changed.", nil
			}
		}
	}

	// if semantic versioning is not enabled, only require versions from the same channel AND with a lower cursor/channel sequence
	allVersions := []*downstreamtypes.DownstreamVersion{}
	if !isSemverRequired {
		for _, v := range appVersions.AllVersions {
			if v.ChannelID == version.ChannelID {
				allVersions = append(allVersions, v)
			}
		}
		downstreamtypes.SortDownstreamVersionsByCursor(allVersions)
	} else {
		allVersions = appVersions.AllVersions
	}

	versionIndex = -1
	for i, v := range allVersions {
		if v.Sequence == version.Sequence {
			versionIndex = i
			break
		}
	}

	deployedVersionIndex = -1
	for i, v := range allVersions {
		if v.Sequence == appVersions.CurrentVersion.Sequence {
			deployedVersionIndex = i
			break
		}
	}

	if deployedVersionIndex == -1 {
		// the deployed version is from a different channel
		return true, "", nil
	}

	// find required versions between the deployed version and the desired version
	requiredVersions := []*downstreamtypes.DownstreamVersion{}
ALL_VERSIONS_LOOP:
	for i, v := range allVersions {
		if !v.IsRequired {
			continue
		}
		if isSameUpstreamRelease(v, version, isSemverRequired) {
			// variants of the same upstream release don't block each other
			continue
		}
		if versionIndex > deployedVersionIndex {
			// this is a past version
			// >= because if the deployed version is required, rolling back isn't allowed
			if i >= deployedVersionIndex && i < versionIndex {
				return false, "One or more non-reversible versions have been deployed since this version.", nil
			}
			continue
		}
		// this is a pending version
		if isSameUpstreamRelease(v, appVersions.CurrentVersion, isSemverRequired) {
			// variants of the deployed upstream release are not required
			continue
		}
		for _, r := range requiredVersions {
			// variants of the same upstream release are only required once
			// since the list is sorted in descending order, the latest variant (highest sequence) will be added first
			if isSameUpstreamRelease(v, r, isSemverRequired) {
				continue ALL_VERSIONS_LOOP
			}
		}
		if i > versionIndex && i < deployedVersionIndex {
			requiredVersions = append(requiredVersions, v)
		}
	}

	if len(requiredVersions) > 0 {
		versionLabels := []string{}
		for _, v := range requiredVersions {
			versionLabels = append([]string{v.VersionLabel}, versionLabels...)
		}
		versionLabelsStr := strings.Join(versionLabels, ", ")
		if len(requiredVersions) == 1 {
			return false, fmt.Sprintf("This version cannot be deployed because version %s is required and must be deployed first.", versionLabelsStr), nil
		}
		return false, fmt.Sprintf("This version cannot be deployed because versions %s are required and must be deployed first.", versionLabelsStr), nil
	}

	return true, "", nil
}

// didECClusterConfigChange compares the embedded cluster config of the version specified to the
// currently deployed version to check if it has changed.
func didECClusterConfigChange(s store.Store, appID string, version *downstreamtypes.DownstreamVersion, currentVersion *downstreamtypes.DownstreamVersion) (bool, error) {
	currentConf, err := s.GetEmbeddedClusterConfigForVersion(appID, currentVersion.Sequence)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get embedded cluster config for current version %d", currentVersion.Sequence)
	}
	currentECConfigBytes, err := json.Marshal(currentConf)
	if err != nil {
		return false, errors.Wrapf(err, "failed to marshal embedded cluster config for current version %d", currentVersion.Sequence)
	}
	thisConf, err := s.GetEmbeddedClusterConfigForVersion(appID, version.Sequence)
	if err != nil {
		return false, errors.Wrap(err, "failed to get embedded cluster config")
	}
	ecConfigBytes, err := json.Marshal(thisConf)
	if err != nil {
		return false, errors.Wrap(err, "failed to marshal embedded cluster config")
	}
	return !bytes.Equal(ecConfigBytes, currentECConfigBytes), nil
}

func getReleaseNotes(appID string, parentSequence int64) (string, error) {
	db := persistence.MustGetDBSession()
	query := `SELECT release_notes FROM app_version WHERE app_id = ? AND sequence = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, parentSequence},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return "", nil
	}

	var releaseNotes gorqlite.NullString
	if err := rows.Scan(&releaseNotes); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return releaseNotes.String, nil
}

func needsKotsUpgrade(app *kotsv1beta1.Application) bool {
	if app == nil {
		return false
	}

	if !kotsutil.IsKotsAutoUpgradeSupported(app) {
		return false
	}

	return !kotsutil.IsKotsVersionCompatibleWithApp(*app, false)
}

func (s *KOTSStore) GetDownstreamOutput(appID string, clusterID string, sequence int64) (*downstreamtypes.DownstreamOutput, error) {
	db := persistence.MustGetDBSession()
	query := `SELECT
	adv.status,
	adv.status_info,
	ado.dryrun_stdout,
	ado.dryrun_stderr,
	ado.apply_stdout,
	ado.apply_stderr,
	ado.helm_stdout,
	ado.helm_stderr
FROM
	app_downstream_version adv
LEFT JOIN
	app_downstream_output ado
ON
	adv.app_id = ado.app_id AND adv.cluster_id = ado.cluster_id AND adv.sequence = ado.downstream_sequence
WHERE
	adv.app_id = ? AND
	adv.cluster_id = ? AND
	adv.sequence = ?`

	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, clusterID, sequence},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return &downstreamtypes.DownstreamOutput{}, nil
	}

	var status gorqlite.NullString
	var statusInfo gorqlite.NullString
	var dryrunStdout gorqlite.NullString
	var dryrunStderr gorqlite.NullString
	var applyStdout gorqlite.NullString
	var applyStderr gorqlite.NullString
	var helmStdout gorqlite.NullString
	var helmStderr gorqlite.NullString

	if err := rows.Scan(&status, &statusInfo, &dryrunStdout, &dryrunStderr, &applyStdout, &applyStderr, &helmStdout, &helmStderr); err != nil {
		return nil, errors.Wrap(err, "failed to select downstream")
	}

	renderError := ""
	if status.String == "failed" {
		renderError = statusInfo.String
	}

	dryrunStdoutDecoded, err := base64.StdEncoding.DecodeString(dryrunStdout.String)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to decode dryrun stdout"))
		dryrunStdoutDecoded = []byte("")
	}

	dryrunStderrDecoded, err := base64.StdEncoding.DecodeString(dryrunStderr.String)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to decode dryrun stderr"))
		dryrunStderrDecoded = []byte("")
	}

	applyStdoutDecoded, err := base64.StdEncoding.DecodeString(applyStdout.String)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to decode apply stdout"))
		applyStdoutDecoded = []byte("")
	}

	applyStderrDecoded, err := base64.StdEncoding.DecodeString(applyStderr.String)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to decode apply stder"))
		applyStderrDecoded = []byte("")
	}

	helmStdoutDecoded, err := base64.StdEncoding.DecodeString(helmStdout.String)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to decode helm stdout"))
		helmStdoutDecoded = []byte("")
	}

	helmStderrDecoded, err := base64.StdEncoding.DecodeString(helmStderr.String)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to decode helm stder"))
		helmStderrDecoded = []byte("")
	}

	output := &downstreamtypes.DownstreamOutput{
		DryrunStdout: string(dryrunStdoutDecoded),
		DryrunStderr: string(dryrunStderrDecoded),
		ApplyStdout:  string(applyStdoutDecoded),
		ApplyStderr:  string(applyStderrDecoded),
		HelmStdout:   string(helmStdoutDecoded),
		HelmStderr:   string(helmStderrDecoded),
		RenderError:  string(renderError),
	}

	return output, nil
}

func (s *KOTSStore) IsDownstreamDeploySuccessful(appID string, clusterID string, sequence int64) (bool, error) {
	db := persistence.MustGetDBSession()

	query := `SELECT is_error
	FROM app_downstream_output
	WHERE app_id = ? AND cluster_id = ? AND downstream_sequence = ?`

	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, clusterID, sequence},
	})
	if err != nil {
		return false, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return false, nil
	}

	var isError bool
	if err := rows.Scan(&isError); err != nil {
		return false, errors.Wrap(err, "failed to select downstream")
	}

	return !isError, nil
}

func (s *KOTSStore) UpdateDownstreamDeployStatus(appID string, clusterID string, sequence int64, isError bool, output downstreamtypes.DownstreamOutput) error {
	db := persistence.MustGetDBSession()

	query := `insert into app_downstream_output (app_id, cluster_id, downstream_sequence, is_error, dryrun_stdout, dryrun_stderr, apply_stdout, apply_stderr, helm_stdout, helm_stderr)
	values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) on conflict (app_id, cluster_id, downstream_sequence) do update set is_error = EXCLUDED.is_error,
	dryrun_stdout = EXCLUDED.dryrun_stdout, dryrun_stderr = EXCLUDED.dryrun_stderr, apply_stdout = EXCLUDED.apply_stdout, apply_stderr = EXCLUDED.apply_stderr,
	helm_stdout = EXCLUDED.helm_stdout, helm_stderr = EXCLUDED.helm_stderr`

	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, clusterID, sequence, isError, output.DryrunStdout, output.DryrunStderr, output.ApplyStdout, output.ApplyStderr, output.HelmStdout, output.HelmStderr},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) DeleteDownstreamDeployStatus(appID string, clusterID string, sequence int64) error {
	db := persistence.MustGetDBSession()

	query := `delete from app_downstream_output where app_id = ? and cluster_id = ? and downstream_sequence = ?`

	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, clusterID, sequence},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}
