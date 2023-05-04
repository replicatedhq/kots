package kotsstore

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/persistence"
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
	"github.com/rqlite/gorqlite"
)

func (s *KOTSStore) SetPreflightProgress(appID string, sequence int64, progress string) error {
	db := persistence.MustGetDBSession()
	query := `update app_downstream_version set preflight_progress = ? where app_id = ? and parent_sequence = ?`

	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{progress, appID, sequence},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) GetPreflightProgress(appID string, sequence int64) (string, error) {
	db := persistence.MustGetDBSession()
	query := `
	SELECT preflight_progress
	FROM app_downstream_version
	WHERE app_id = ? AND sequence = ?`

	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, sequence},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return "", ErrNotFound
	}

	var progress gorqlite.NullString
	if err := rows.Scan(&progress); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return progress.String, nil
}

func (s *KOTSStore) SetPreflightResults(appID string, sequence int64, results []byte) error {
	db := persistence.MustGetDBSession()
	query := `update app_downstream_version set preflight_result = ?, preflight_result_created_at = ?,
status = (case when status = 'deployed' then 'deployed' when status = 'deploying' then 'deploying' else 'pending' end),
preflight_progress = NULL, preflight_skipped = false
where app_id = ? and parent_sequence = ?`

	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{string(results), time.Now().Unix(), appID, sequence},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) GetPreflightResults(appID string, sequence int64) (*preflighttypes.PreflightResult, error) {
	db := persistence.MustGetDBSession()
	query := `
	SELECT
		app_downstream_version.preflight_result,
		app_downstream_version.preflight_result_created_at,
		app_downstream_version.preflight_skipped,
		app.slug as app_slug,
		cluster.slug as cluster_slug,
		app_version.preflight_spec 
	FROM app_downstream_version
		INNER JOIN app ON app_downstream_version.app_id = app.id
		INNER JOIN cluster ON app_downstream_version.cluster_id = cluster.id
		INNER JOIN app_version ON app_downstream_version.app_id = app_version.app_id AND app_downstream_version.parent_sequence = app_version.sequence
	WHERE
		app_downstream_version.app_id = ? AND
		app_downstream_version.sequence = ?`

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

	r, err := s.preflightResultFromRow(rows)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get preflight result from row")
	}

	return r, nil
}

func (s *KOTSStore) ResetPreflightResults(appID string, sequence int64) error {
	db := persistence.MustGetDBSession()
	query := `update app_downstream_version set preflight_result=null, preflight_result_created_at=null, preflight_skipped=false where app_id = ? and parent_sequence = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, sequence},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}
	return nil
}

func (s *KOTSStore) SetIgnorePreflightPermissionErrors(appID string, sequence int64) error {
	db := persistence.MustGetDBSession()
	query := `UPDATE app_downstream_version
	SET status = 'pending_preflight', preflight_ignore_permissions = true, preflight_result = null, preflight_skipped = false
	WHERE app_id = ? AND sequence = ?`

	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, sequence},
	})
	if err != nil {
		return fmt.Errorf("failed to set downstream version ignore rbac errors: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) preflightResultFromRow(row gorqlite.QueryResult) (*preflighttypes.PreflightResult, error) {
	r := &preflighttypes.PreflightResult{}

	var preflightResult gorqlite.NullString
	var preflightResultCreatedAt gorqlite.NullTime
	var preflightSpec gorqlite.NullString

	if err := row.Scan(
		&preflightResult,
		&preflightResultCreatedAt,
		&r.Skipped,
		&r.AppSlug,
		&r.ClusterSlug,
		&preflightSpec,
	); err != nil {
		return nil, errors.Wrap(err, "failed to scan")
	}

	r.Result = preflightResult.String
	if preflightResultCreatedAt.Valid {
		r.CreatedAt = &preflightResultCreatedAt.Time
	}

	var err error
	r.HasFailingStrictPreflights, err = s.hasFailingStrictPreflights(preflightSpec, preflightResult)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check for failing strict preflight")
	}

	return r, nil
}

func (s *KOTSStore) hasFailingStrictPreflights(preflightSpecStr gorqlite.NullString, preflightResultStr gorqlite.NullString) (bool, error) {
	hasFailingStrictPreflights, err := s.hasStrictPreflights(preflightSpecStr)
	if err != nil {
		return false, errors.Wrap(err, "failed to check for strict preflight")
	}

	if preflightResultStr.Valid && preflightResultStr.String != "" {
		preflightResult := troubleshootpreflight.UploadPreflightResults{}
		if err := json.Unmarshal([]byte(preflightResultStr.String), &preflightResult); err != nil {
			return false, errors.Wrap(err, "failed to unmarshal preflightResults")
		}
		hasFailingStrictPreflights = hasFailingStrictPreflights && troubleshootpreflight.HasStrictAnalyzersFailed(&preflightResult)
	}
	return hasFailingStrictPreflights, nil
}

func (s *KOTSStore) hasStrictPreflights(preflightSpecStr gorqlite.NullString) (bool, error) {
	hasStrictPreflights := false
	if preflightSpecStr.Valid && preflightSpecStr.String != "" {
		preflight, err := kotsutil.LoadPreflightFromContents([]byte(preflightSpecStr.String))
		if err != nil {
			return false, errors.Wrap(err, "failed to load preflights from spec")
		}
		hasStrictPreflights, err = troubleshootpreflight.HasStrictAnalyzers(preflight)
		if err != nil {
			return false, errors.Wrap(err, "failed to check strict preflights from spec")
		}
	}
	return hasStrictPreflights, nil
}
