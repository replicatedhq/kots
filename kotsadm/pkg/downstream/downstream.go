package downstream

import (
	"database/sql"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

type scannable interface {
	Scan(dest ...interface{}) error
}

func ListDownstreamsForApp(appID string) ([]*types.Downstream, error) {
	db := persistence.MustGetPGSession()
	query := `select c.id, c.slug, d.downstream_name, d.current_sequence from app_downstream d inner join cluster c on d.cluster_id = c.id where app_id = $1`
	rows, err := db.Query(query, appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get downstreams")
	}
	defer rows.Close()

	downstreams := []*types.Downstream{}
	for rows.Next() {
		downstream := types.Downstream{
			CurrentSequence: -1,
		}
		var sequence sql.NullInt64
		if err := rows.Scan(&downstream.ClusterID, &downstream.ClusterSlug, &downstream.Name, &sequence); err != nil {
			return nil, errors.Wrap(err, "failed to scan downstream")
		}
		if sequence.Valid {
			downstream.CurrentSequence = sequence.Int64
		}

		downstreams = append(downstreams, &downstream)
	}

	return downstreams, nil
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

// GetDownstreamVersionStatus gets the status for the downstream version with the given sequence and app id
func GetDownstreamVersionStatus(appID string, sequence int64) (string, error) {
	db := persistence.MustGetPGSession()
	query := `select status from app_downstream_version where app_id = $1 and sequence = $2`
	row := db.QueryRow(query, appID, sequence)
	var status sql.NullString
	err := row.Scan(&status)
	if err != nil {
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

func SetIgnorePreflightPermissionErrors(appID string, sequence int64) error {
	db := persistence.MustGetPGSession()
	query := `UPDATE app_downstream_version
	SET status = 'pending_preflight', preflight_ignore_permissions = true, preflight_result = null
	WHERE app_id = $1 AND sequence = $2`

	_, err := db.Exec(query, appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to set downstream version ignore rbac errors")
	}

	return nil
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
	s := "unknown";

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

	return s;
}
