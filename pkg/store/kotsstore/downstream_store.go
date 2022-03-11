package kotsstore

import (
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/store/types"
	"k8s.io/client-go/kubernetes/scheme"
)

func (s *KOTSStore) GetCurrentSequence(appID string, clusterID string) (int64, error) {
	db := persistence.MustGetDBSession()
	query := `select current_sequence from app_downstream where app_id = $1 and cluster_id = $2`
	row := db.QueryRow(query, appID, clusterID)

	var currentSequence sql.NullInt64
	if err := row.Scan(&currentSequence); err != nil {
		return -1, errors.Wrap(err, "failed to scan")
	}

	if !currentSequence.Valid {
		return -1, nil
	}

	return currentSequence.Int64, nil
}

func (s *KOTSStore) GetCurrentParentSequence(appID string, clusterID string) (int64, error) {
	currentSequence, err := s.GetCurrentSequence(appID, clusterID)
	if err != nil {
		return -1, errors.Wrap(err, "failed to get current parent sequence")
	}
	if currentSequence == -1 {
		return -1, nil
	}

	db := persistence.MustGetDBSession()
	query := `select parent_sequence from app_downstream_version where app_id = $1 and cluster_id = $2 and sequence = $3`
	row := db.QueryRow(query, appID, clusterID, currentSequence)

	var parentSequence sql.NullInt64
	if err := row.Scan(&parentSequence); err != nil {
		return -1, errors.Wrap(err, "failed to scan")
	}

	if !parentSequence.Valid {
		return -1, nil
	}

	return parentSequence.Int64, nil
}

func (s *KOTSStore) GetParentSequenceForSequence(appID string, clusterID string, sequence int64) (int64, error) {
	db := persistence.MustGetDBSession()
	query := `select parent_sequence from app_downstream_version where app_id = $1 and cluster_id = $2 and sequence = $3`
	row := db.QueryRow(query, appID, clusterID, sequence)

	var parentSequence sql.NullInt64
	if err := row.Scan(&parentSequence); err != nil {
		return -1, errors.Wrap(err, "failed to scan")
	}

	if !parentSequence.Valid {
		return -1, nil
	}

	return parentSequence.Int64, nil
}

func (s *KOTSStore) GetPreviouslyDeployedSequence(appID string, clusterID string) (int64, error) {
	db := persistence.MustGetDBSession()
	query := `select sequence from app_downstream_version where app_id = $1 and cluster_id = $2 and applied_at is not null order by applied_at desc limit 2`
	rows, err := db.Query(query, appID, clusterID)
	if err != nil {
		return -1, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()

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

// SetDownstreamVersionReady sets the status for the downstream version with the given sequence and app id to "pending"
func (s *KOTSStore) SetDownstreamVersionReady(appID string, sequence int64) error {
	db := persistence.MustGetDBSession()
	query := `update app_downstream_version set status = 'pending' where app_id = $1 and sequence = $2`
	_, err := db.Exec(query, appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to set downstream version ready")
	}

	return nil
}

// SetDownstreamVersionPendingPreflight sets the status for the downstream version with the given sequence and app id to "pending_preflight"
func (s *KOTSStore) SetDownstreamVersionPendingPreflight(appID string, sequence int64) error {
	db := persistence.MustGetDBSession()
	query := `update app_downstream_version set status = 'pending_preflight' where app_id = $1 and sequence = $2`
	_, err := db.Exec(query, appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to set downstream version pending preflight")
	}

	return nil
}

// UpdateDownstreamVersionStatus updates the status and status info for the downstream version with the given sequence and app id
func (s *KOTSStore) UpdateDownstreamVersionStatus(appID string, sequence int64, status string, statusInfo string) error {
	db := persistence.MustGetDBSession()
	query := `update app_downstream_version set status = $1, status_info = $2 where app_id = $3 and sequence = $4`
	_, err := db.Exec(query, status, statusInfo, appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}

// GetDownstreamVersionStatus gets the status for the downstream version with the given sequence and app id
func (s *KOTSStore) GetDownstreamVersionStatus(appID string, sequence int64) (types.DownstreamVersionStatus, error) {
	db := persistence.MustGetDBSession()
	query := `select status from app_downstream_version where app_id = $1 and sequence = $2`
	row := db.QueryRow(query, appID, sequence)
	var status sql.NullString
	err := row.Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", errors.Wrap(err, "failed to get downstream version")
	}

	return types.DownstreamVersionStatus(status.String), nil
}

func (s *KOTSStore) GetIgnoreRBACErrors(appID string, sequence int64) (bool, error) {
	db := persistence.MustGetDBSession()
	query := `SELECT preflight_ignore_permissions FROM app_downstream_version
	WHERE app_id = $1 and sequence = $2 LIMIT 1`
	row := db.QueryRow(query, appID, sequence)

	var shouldIgnore sql.NullBool
	if err := row.Scan(&shouldIgnore); err != nil {
		return false, errors.Wrap(err, "failed to select downstream")
	}

	if !shouldIgnore.Valid {
		return false, nil
	}

	return shouldIgnore.Bool, nil
}

func (s *KOTSStore) GetLatestDownstreamVersion(appID string, clusterID string, downloadedOnly bool) (*downstreamtypes.DownstreamVersion, error) {
	downstreamVersions, err := s.GetAppVersions(appID, clusterID, downloadedOnly)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find app versions")
	}
	if len(downstreamVersions.AllVersions) == 0 {
		return nil, errors.New("no app versions found")
	}
	return downstreamVersions.AllVersions[0], nil
}

func (s *KOTSStore) GetCurrentVersion(appID string, clusterID string) (*downstreamtypes.DownstreamVersion, error) {
	currentSequence, err := s.GetCurrentSequence(appID, clusterID)
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
	adv.diff_summary,
	adv.diff_summary_error,
	adv.preflight_result,
	adv.preflight_result_created_at,
	adv.preflight_skipped,
	adv.git_commit_url,
	adv.git_deployable,
	ado.is_error,
	av.upstream_released_at,
	av.kots_installation_spec,
	av.kots_app_spec,
	av.version_label,
	av.preflight_spec
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
	 adv.app_id = $1 AND
	 adv.cluster_id = $2 AND
	 adv.sequence = $3`
	row := db.QueryRow(query, appID, clusterID, currentSequence)

	v, err := s.downstreamVersionFromRow(appID, row)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get version from row")
	}

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
 	adv.app_id = $1 AND adv.cluster_id = $2 AND adv.sequence = $3`
	row := db.QueryRow(query, appID, clusterID, sequence)

	var status sql.NullString
	var hasError sql.NullBool
	if err := row.Scan(&status, &hasError); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}
	versionStatus := getDownstreamVersionStatus(types.DownstreamVersionStatus(status.String), hasError)

	return types.DownstreamVersionStatus(versionStatus), nil
}

func (s *KOTSStore) GetAppVersions(appID string, clusterID string, downloadedOnly bool) (*downstreamtypes.DownstreamVersions, error) {
	currentVersion, err := s.GetCurrentVersion(appID, clusterID)
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
	adv.diff_summary,
	adv.diff_summary_error,
	adv.preflight_result,
	adv.preflight_result_created_at,
	adv.preflight_skipped,
	adv.git_commit_url,
	adv.git_deployable,
	ado.is_error,
	av.upstream_released_at,
	av.kots_installation_spec,
	av.kots_app_spec,
	av.version_label,
	av.preflight_spec
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
	 adv.app_id = $1 AND
	 adv.cluster_id = $2`

	if downloadedOnly {
		query += fmt.Sprintf(` AND adv.status != '%s'`, types.VersionPendingDownload)
	}

	query += ` ORDER BY adv.sequence DESC`

	rows, err := db.Query(query, appID, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()

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
	downstreamtypes.SortDownstreamVersions(result, license.Spec.IsSemverRequired)

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

func (s *KOTSStore) FindAppVersions(appID string, downloadedOnly bool) (*downstreamtypes.DownstreamVersions, error) {
	downstreams, err := s.ListDownstreamsForApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app downstreams")
	}
	if len(downstreams) == 0 {
		return nil, errors.New("app has no downstreams")
	}

	for _, d := range downstreams {
		clusterID := d.ClusterID
		downstreamVersions, err := s.GetAppVersions(appID, clusterID, downloadedOnly)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get downstream versions for cluster %s", clusterID)
		}
		if len(downstreamVersions.AllVersions) > 0 {
			return downstreamVersions, nil
		}
	}

	return nil, errors.New("app has no versions")
}

func (s *KOTSStore) downstreamVersionFromRow(appID string, row scannable) (*downstreamtypes.DownstreamVersion, error) {
	v := &downstreamtypes.DownstreamVersion{}

	var createdOn persistence.NullStringTime
	var versionLabel sql.NullString
	var status sql.NullString
	var parentSequence sql.NullInt64
	var deployedAt persistence.NullStringTime
	var source sql.NullString
	var diffSummary sql.NullString
	var diffSummaryError sql.NullString
	var preflightResult sql.NullString
	var preflightResultCreatedAt persistence.NullStringTime
	var preflightSkipped sql.NullBool
	var preflightSpecStr sql.NullString
	var commitURL sql.NullString
	var gitDeployable sql.NullBool
	var hasError sql.NullBool
	var upstreamReleasedAt persistence.NullStringTime
	var kotsInstallationSpecStr sql.NullString
	var kotsAppSpecStr sql.NullString

	if err := row.Scan(
		&createdOn,
		&status,
		&v.Sequence,
		&parentSequence,
		&deployedAt,
		&source,
		&diffSummary,
		&diffSummaryError,
		&preflightResult,
		&preflightResultCreatedAt,
		&preflightSkipped,
		&commitURL,
		&gitDeployable,
		&hasError,
		&upstreamReleasedAt,
		&kotsInstallationSpecStr,
		&kotsAppSpecStr,
		&versionLabel,
		&preflightSpecStr,
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

	v.Status = getDownstreamVersionStatus(types.DownstreamVersionStatus(status.String), hasError)
	v.ParentSequence = parentSequence.Int64

	if deployedAt.Valid {
		v.DeployedAt = &deployedAt.Time
	}
	v.Source = source.String
	v.DiffSummary = diffSummary.String
	v.DiffSummaryError = diffSummaryError.String
	v.PreflightResult = preflightResult.String

	if preflightResultCreatedAt.Valid {
		v.PreflightResultCreatedAt = &preflightResultCreatedAt.Time
	}
	v.PreflightSkipped = preflightSkipped.Bool
	v.CommitURL = commitURL.String
	v.GitDeployable = gitDeployable.Bool

	releaseNotes, err := getReleaseNotes(appID, v.ParentSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get release notes")
	}
	v.ReleaseNotes = releaseNotes

	if upstreamReleasedAt.Valid {
		v.UpstreamReleasedAt = &upstreamReleasedAt.Time
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	if kotsInstallationSpecStr.String != "" {
		obj, _, err := decode([]byte(kotsInstallationSpecStr.String), nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode installation spec yaml")
		}
		installationSpec := obj.(*kotsv1beta1.Installation)

		v.YamlErrors = installationSpec.Spec.YAMLErrors
	}

	if v.Status == types.VersionPendingDownload {
		downloadTaskID := fmt.Sprintf("update-download.%d", v.Sequence)
		downloadStatus, downloadStatusMessage, err := s.GetTaskStatus(downloadTaskID)
		if err != nil {
			// don't fail on this
			logger.Error(errors.Wrap(err, fmt.Sprintf("failed to get %s task status", downloadTaskID)))
		}
		v.DownloadStatus = downstreamtypes.DownloadStatus{
			Message: downloadStatusMessage,
			Status:  downloadStatus,
		}
	}

	if kotsAppSpecStr.String != "" {
		obj, _, err := decode([]byte(kotsAppSpecStr.String), nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode kots app spec yaml")
		}
		v.KotsApplication = obj.(*kotsv1beta1.Application)
	}

	v.NeedsKotsUpgrade = needsKotsUpgrade(v.KotsApplication)
	v.HasFailingStrictPreflights, err = s.hasFailingStrictPreflights(preflightSpecStr, preflightResult)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get strict preflight results")
	}

	return v, nil
}

func getReleaseNotes(appID string, parentSequence int64) (string, error) {
	db := persistence.MustGetDBSession()
	query := `SELECT release_notes FROM app_version WHERE app_id = $1 AND sequence = $2`
	row := db.QueryRow(query, appID, parentSequence)

	var releaseNotes sql.NullString
	if err := row.Scan(&releaseNotes); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", errors.Wrap(err, "failed to scan")
	}

	return releaseNotes.String, nil
}

func getDownstreamVersionStatus(status types.DownstreamVersionStatus, hasError sql.NullBool) types.DownstreamVersionStatus {
	s := types.VersionUnknown

	// first check if operator has reported back.
	// and if it hasn't, we should not show "deployed" to the user.

	if hasError.Valid && !hasError.Bool {
		s = status
	} else if hasError.Valid && hasError.Bool {
		s = types.VersionFailed
	} else if status == types.VersionDeployed {
		s = types.VersionDeploying
	} else if status != types.DownstreamVersionStatus("") {
		s = status
	}

	return s
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
	adv.app_id = $1 AND
	adv.cluster_id = $2 AND
	adv.sequence = $3`
	row := db.QueryRow(query, appID, clusterID, sequence)

	var status sql.NullString
	var statusInfo sql.NullString
	var dryrunStdout sql.NullString
	var dryrunStderr sql.NullString
	var applyStdout sql.NullString
	var applyStderr sql.NullString
	var helmStdout sql.NullString
	var helmStderr sql.NullString

	if err := row.Scan(&status, &statusInfo, &dryrunStdout, &dryrunStderr, &applyStdout, &applyStderr, &helmStdout, &helmStderr); err != nil {
		if err == sql.ErrNoRows {
			return &downstreamtypes.DownstreamOutput{}, nil
		}
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
	WHERE app_id = $1 AND cluster_id = $2 AND downstream_sequence = $3`

	row := db.QueryRow(query, appID, clusterID, sequence)

	var isError bool
	if err := row.Scan(&isError); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to select downstream")
	}

	return !isError, nil
}

func (s *KOTSStore) UpdateDownstreamDeployStatus(appID string, clusterID string, sequence int64, isError bool, output downstreamtypes.DownstreamOutput) error {
	db := persistence.MustGetDBSession()

	query := `insert into app_downstream_output (app_id, cluster_id, downstream_sequence, is_error, dryrun_stdout, dryrun_stderr, apply_stdout, apply_stderr, helm_stdout, helm_stderr)
	values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) on conflict (app_id, cluster_id, downstream_sequence) do update set is_error = EXCLUDED.is_error,
	dryrun_stdout = EXCLUDED.dryrun_stdout, dryrun_stderr = EXCLUDED.dryrun_stderr, apply_stdout = EXCLUDED.apply_stdout, apply_stderr = EXCLUDED.apply_stderr,
	helm_stdout = EXCLUDED.helm_stdout, helm_stderr = EXCLUDED.helm_stderr`

	_, err := db.Exec(query, appID, clusterID, sequence, isError, output.DryrunStdout, output.DryrunStderr, output.ApplyStdout, output.ApplyStderr, output.HelmStdout, output.HelmStderr)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}

func (s *KOTSStore) DeleteDownstreamDeployStatus(appID string, clusterID string, sequence int64) error {
	db := persistence.MustGetDBSession()

	query := `delete from app_downstream_output where app_id = $1 and cluster_id = $2 and downstream_sequence = $3`

	_, err := db.Exec(query, appID, clusterID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}
