package downstream

import (
	"database/sql"
	"encoding/base64"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

type scannable interface {
	Scan(dest ...interface{}) error
}

func GetClusterIDFromDeployToken(token string) (string, error) {
	db := persistence.MustGetPGSession()
	query := `select id from cluster where token = $1`
	row := db.QueryRow(query, token)

	var clusterID string
	if err := row.Scan(&clusterID); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return clusterID, nil
}

func GetCurrentSequence(appID string, clusterID string) (int64, error) {
	db := persistence.MustGetPGSession()
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

func GetCurrentParentSequence(appID string, clusterID string) (int64, error) {
	currentSequence, err := GetCurrentSequence(appID, clusterID)
	if err != nil {
		return -1, errors.Wrap(err, "failed to get current sequence")
	}
	if currentSequence == -1 {
		return -1, nil
	}

	db := persistence.MustGetPGSession()
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

func GetParentSequenceForSequence(appID string, clusterID string, sequence int64) (int64, error) {
	db := persistence.MustGetPGSession()
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

func GetPreviouslyDeployedSequence(appID string, clusterID string) (int64, error) {
	db := persistence.MustGetPGSession()
	query := `select sequence from app_downstream_version where app_id = $1 and cluster_id = $2 and applied_at is not null order by applied_at desc limit 2`
	rows, err := db.Query(query, appID, clusterID)
	if err != nil {
		return -1, errors.Wrap(err, "failed to query")
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

// SetDownstreamVersionReady sets the status for the downstream version with the given sequence and app id to "pending"
func SetDownstreamVersionReady(appID string, sequence int64) error {
	db := persistence.MustGetPGSession()
	query := `update app_downstream_version set status = 'pending' where app_id = $1 and sequence = $2`
	_, err := db.Exec(query, appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to set downstream version ready")
	}

	return nil
}

// SetDownstreamVersionPendingPreflight sets the status for the downstream version with the given sequence and app id to "pending_preflight"
func SetDownstreamVersionPendingPreflight(appID string, sequence int64) error {
	db := persistence.MustGetPGSession()
	query := `update app_downstream_version set status = 'pending_preflight' where app_id = $1 and sequence = $2`
	_, err := db.Exec(query, appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to set downstream version pending preflight")
	}

	return nil
}

// UpdateDownstreamStatus updates the status and status info for the downstream version with the given sequence and app id
func UpdateDownstreamStatus(appID string, sequence int64, status string, statusInfo string) error {
	db := persistence.MustGetPGSession()
	query := `update app_downstream_version set status = $3, status_info = $4 where app_id = $1 and sequence = $2`
	_, err := db.Exec(query, appID, sequence, status, statusInfo)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}

// GetDownstreamVersionStatus gets the status for the downstream version with the given sequence and app id
func GetDownstreamVersionStatus(appID string, sequence int64) (string, error) {
	db := persistence.MustGetPGSession()
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

	return status.String, nil
}

func GetIgnoreRBACErrors(appID string, sequence int64) (bool, error) {
	db := persistence.MustGetPGSession()
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

func GetCurrentVersion(appID string, clusterID string) (*types.DownstreamVersion, error) {
	currentSequence, err := GetCurrentSequence(appID, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current sequence")
	}
	if currentSequence == -1 {
		return nil, nil
	}

	db := persistence.MustGetPGSession()
	query := `SELECT
	adv.created_at,
	adv.version_label,
	adv.status,
	adv.sequence,
	adv.parent_sequence,
	adv.applied_at,
	adv.source,
	adv.diff_summary,
	adv.diff_summary_error,
	adv.preflight_result,
	adv.preflight_result_created_at,
	adv.git_commit_url,
	adv.git_deployable,
	ado.is_error,
	av.kots_installation_spec
 FROM
	 app_downstream_version AS adv
 LEFT JOIN
	 app_version AS av
 ON
	 adv.app_id = av.app_id AND adv.sequence = av.sequence
 LEFT JOIN
	 app_downstream_output AS ado
 ON
	 adv.app_id = ado.app_id AND adv.cluster_id = ado.cluster_id AND adv.sequence = ado.downstream_sequence
 WHERE
	 adv.app_id = $1 AND
	 adv.cluster_id = $3 AND
	 adv.sequence = $2
 ORDER BY
	 adv.sequence DESC`
	row := db.QueryRow(query, appID, currentSequence, clusterID)

	v, err := versionFromRow(appID, row)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get version from row")
	}

	return v, nil
}

func GetPendingVersions(appID string, clusterID string) ([]types.DownstreamVersion, error) {
	currentSequence, err := GetCurrentSequence(appID, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current sequence")
	}

	db := persistence.MustGetPGSession()
	query := `SELECT
	adv.created_at,
	adv.version_label,
	adv.status,
	adv.sequence,
	adv.parent_sequence,
	adv.applied_at,
	adv.source,
	adv.diff_summary,
	adv.diff_summary_error,
	adv.preflight_result,
	adv.preflight_result_created_at,
	adv.git_commit_url,
	adv.git_deployable,
	ado.is_error,
	av.kots_installation_spec
 FROM
	 app_downstream_version AS adv
 LEFT JOIN
	 app_version AS av
 ON
	 adv.app_id = av.app_id AND adv.sequence = av.sequence
 LEFT JOIN
	 app_downstream_output AS ado
 ON
	 adv.app_id = ado.app_id AND adv.cluster_id = ado.cluster_id AND adv.sequence = ado.downstream_sequence
 WHERE
	 adv.app_id = $1 AND
	 adv.cluster_id = $3 AND
	 adv.sequence > $2
 ORDER BY
	 adv.sequence DESC`

	rows, err := db.Query(query, appID, currentSequence, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()

	versions := []types.DownstreamVersion{}
	for rows.Next() {
		v, err := versionFromRow(appID, rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get version from row")
		}
		if v != nil {
			versions = append(versions, *v)
		}
	}

	return versions, nil
}

func GetPastVersions(appID string, clusterID string) ([]types.DownstreamVersion, error) {
	currentSequence, err := GetCurrentSequence(appID, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current sequence")
	}
	if currentSequence == -1 {
		return []types.DownstreamVersion{}, nil
	}

	db := persistence.MustGetPGSession()
	query := `SELECT
	adv.created_at,
	adv.version_label,
	adv.status,
	adv.sequence,
	adv.parent_sequence,
	adv.applied_at,
	adv.source,
	adv.diff_summary,
	adv.diff_summary_error,
	adv.preflight_result,
	adv.preflight_result_created_at,
	adv.git_commit_url,
	adv.git_deployable,
	ado.is_error,
	av.kots_installation_spec
 FROM
	 app_downstream_version AS adv
 LEFT JOIN
	 app_version AS av
 ON
	 adv.app_id = av.app_id AND adv.sequence = av.sequence
 LEFT JOIN
	 app_downstream_output AS ado
 ON
	 adv.app_id = ado.app_id AND adv.cluster_id = ado.cluster_id AND adv.sequence = ado.downstream_sequence
 WHERE
	 adv.app_id = $1 AND
	 adv.cluster_id = $3 AND
	 adv.sequence < $2
 ORDER BY
	 adv.sequence DESC`

	rows, err := db.Query(query, appID, currentSequence, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()

	versions := []types.DownstreamVersion{}
	for rows.Next() {
		v, err := versionFromRow(appID, rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get version from row")
		}
		if v != nil {
			versions = append(versions, *v)
		}
	}

	return versions, nil
}

func versionFromRow(appID string, row scannable) (*types.DownstreamVersion, error) {
	v := &types.DownstreamVersion{}

	var createdOn sql.NullTime
	var versionLabel sql.NullString
	var status sql.NullString
	var parentSequence sql.NullInt64
	var deployedAt sql.NullTime
	var source sql.NullString
	var diffSummary sql.NullString
	var diffSummaryError sql.NullString
	var preflightResult sql.NullString
	var preflightResultCreatedAt sql.NullTime
	var commitURL sql.NullString
	var gitDeployable sql.NullBool
	var hasError sql.NullBool
	var kotsInstallationSpecStr sql.NullString

	if err := row.Scan(
		&createdOn,
		&versionLabel,
		&status,
		&v.Sequence,
		&parentSequence,
		&deployedAt,
		&source,
		&diffSummary,
		&diffSummaryError,
		&preflightResult,
		&preflightResultCreatedAt,
		&commitURL,
		&gitDeployable,
		&hasError,
		&kotsInstallationSpecStr,
	); err != nil {
		return nil, errors.Wrap(err, "failed to scan")
	}

	if createdOn.Valid {
		v.CreatedOn = &createdOn.Time
	}
	v.VersionLabel = versionLabel.String
	v.Status = getStatus(status.String, hasError)
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
	v.CommitURL = commitURL.String
	v.GitDeployable = gitDeployable.Bool

	releaseNotes, err := getReleaseNotes(appID, v.ParentSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get release notes")
	}
	v.ReleaseNotes = releaseNotes

	if kotsInstallationSpecStr.Valid && kotsInstallationSpecStr.String != "" {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(kotsInstallationSpecStr.String), nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode installation spec yaml")
		}
		installationSpec := obj.(*kotsv1beta1.Installation)

		v.YamlErrors = installationSpec.Spec.YAMLErrors
	}

	return v, nil
}

func getReleaseNotes(appID string, parentSequence int64) (string, error) {
	db := persistence.MustGetPGSession()
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

func getStatus(status string, hasError sql.NullBool) string {
	s := "unknown"

	// first check if operator has reported back.
	// and if it hasn't, we should not show "deployed" to the user.

	if hasError.Valid && !hasError.Bool {
		s = status
	} else if hasError.Valid && hasError.Bool {
		s = "failed"
	} else if status == "deployed" {
		s = "deploying"
	} else if status != "" {
		s = status
	}

	return s
}

func GetDownstreamOutput(appID string, clusterID string, sequence int64) (*types.DownstreamOutput, error) {
	db := persistence.MustGetPGSession()
	query := `SELECT
	adv.status,
	adv.status_info,
	ado.dryrun_stdout,
	ado.dryrun_stderr,
	ado.apply_stdout,
	ado.apply_stderr
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

	if err := row.Scan(&status, &statusInfo, &dryrunStdout, &dryrunStderr, &applyStdout, &applyStderr); err != nil {
		if err == sql.ErrNoRows {
			return &types.DownstreamOutput{}, nil
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

	output := &types.DownstreamOutput{
		DryrunStdout: string(dryrunStdoutDecoded),
		DryrunStderr: string(dryrunStderrDecoded),
		ApplyStdout:  string(applyStdoutDecoded),
		ApplyStderr:  string(applyStderrDecoded),
		RenderError:  string(renderError),
	}

	return output, nil
}

func IsDownstreamDeploySuccessful(appID string, clusterID string, sequence int64) (bool, error) {
	db := persistence.MustGetPGSession()

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

func UpdateDownstreamDeployStatus(appID string, clusterID string, sequence int64, isError bool, output types.DownstreamOutput) error {
	db := persistence.MustGetPGSession()

	query := `insert into app_downstream_output (app_id, cluster_id, downstream_sequence, is_error, dryrun_stdout, dryrun_stderr, apply_stdout, apply_stderr)
	values ($1, $2, $3, $4, $5, $6, $7, $8) on conflict (app_id, cluster_id, downstream_sequence) do update set is_error = EXCLUDED.is_error,
	dryrun_stdout = EXCLUDED.dryrun_stdout, dryrun_stderr = EXCLUDED.dryrun_stderr, apply_stdout = EXCLUDED.apply_stdout, apply_stderr = EXCLUDED.apply_stderr`

	_, err := db.Exec(query, appID, clusterID, sequence, isError, output.DryrunStdout, output.DryrunStderr, output.ApplyStdout, output.ApplyStderr)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}
