package s3pg

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"
	preflighttypes "github.com/replicatedhq/kots/kotsadm/pkg/preflight/types"
	"github.com/replicatedhq/kots/pkg/persistence"
)

func (c S3PGStore) SetPreflightResults(appID string, sequence int64, results []byte) error {
	db := persistence.MustGetPGSession()
	query := `update app_downstream_version set preflight_result = $1, preflight_result_created_at = $2,
status = (case when status = 'deployed' then 'deployed' else 'pending' end)
where app_id = $3 and parent_sequence = $4`

	_, err := db.Exec(query, results, time.Now(), appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to write preflight results")
	}

	return nil
}

func (c S3PGStore) GetPreflightResults(appID string, sequence int64) (*preflighttypes.PreflightResult, error) {
	db := persistence.MustGetPGSession()
	query := `
	SELECT
		app_downstream_version.preflight_result,
		app_downstream_version.preflight_result_created_at,
		app.slug as app_slug,
		cluster.slug as cluster_slug
	FROM app_downstream_version
		INNER JOIN app ON app_downstream_version.app_id = app.id
		INNER JOIN cluster ON app_downstream_version.cluster_id = cluster.id
	WHERE
		app_downstream_version.app_id = $1 AND
		app_downstream_version.sequence = $2`

	row := db.QueryRow(query, appID, sequence)
	r, err := preflightResultFromRow(row)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get preflight result from row")
	}

	return r, nil
}

func (c S3PGStore) GetLatestPreflightResultsForSequenceZero() (*preflighttypes.PreflightResult, error) {
	db := persistence.MustGetPGSession()
	query := `
	SELECT
		app_downstream_version.preflight_result,
		app_downstream_version.preflight_result_created_at,
		app.slug as app_slug,
		cluster.slug as cluster_slug
	FROM app_downstream_version
		INNER JOIN (
			SELECT id, slug FROM app WHERE current_sequence = 0 ORDER BY created_at DESC LIMIT 1
		) AS app ON app_downstream_version.app_id = app.id
		INNER JOIN cluster ON app_downstream_version.cluster_id = cluster.id
	WHERE
		app_downstream_version.sequence = 0`

	row := db.QueryRow(query)
	r, err := preflightResultFromRow(row)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get preflight result from row")
	}

	return r, nil
}

func (s S3PGStore) ResetPreflightResults(appID string, sequence int64) error {
	db := persistence.MustGetPGSession()
	query := `update app_downstream_version set preflight_result=null, preflight_result_created_at=null where app_id = $1 and parent_sequence = $2`
	_, err := db.Exec(query, appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s S3PGStore) SetIgnorePreflightPermissionErrors(appID string, sequence int64) error {
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

func preflightResultFromRow(row scannable) (*preflighttypes.PreflightResult, error) {
	r := &preflighttypes.PreflightResult{}

	var preflightResult sql.NullString
	var preflightResultCreatedAt sql.NullTime

	if err := row.Scan(
		&preflightResult,
		&preflightResultCreatedAt,
		&r.AppSlug,
		&r.ClusterSlug,
	); err != nil {
		return nil, errors.Wrap(err, "failed to scan")
	}

	r.Result = preflightResult.String
	if preflightResultCreatedAt.Valid {
		r.CreatedAt = &preflightResultCreatedAt.Time
	}

	return r, nil
}
