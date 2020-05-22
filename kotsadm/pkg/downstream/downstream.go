package downstream

import (
	"database/sql"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
)

func Get(appID string, clusterID string) (*types.Downstream, error) {
	db := persistence.MustGetPGSession()
	query := `select cluster_id, downstream_name, current_sequence from app_downstream where app_id = $1 and cluster_id = $2`
	row := db.QueryRow(query, appID, clusterID)

	downstream := types.Downstream{
		CurrentSequence: -1,
	}
	var sequence sql.NullInt64
	if err := row.Scan(&downstream.ClusterID, &downstream.Name, &sequence); err != nil {
		return nil, errors.Wrap(err, "failed to scan downstream")
	}
	if sequence.Valid {
		downstream.CurrentSequence = sequence.Int64
	}

	return &downstream, nil
}

func ListDownstreamsForApp(appID string) ([]*types.Downstream, error) {
	db := persistence.MustGetPGSession()
	query := `select cluster_id, downstream_name, current_sequence from app_downstream where app_id = $1`
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
		if err := rows.Scan(&downstream.ClusterID, &downstream.Name, &sequence); err != nil {
			return nil, errors.Wrap(err, "failed to scan downstream")
		}
		if sequence.Valid {
			downstream.CurrentSequence = sequence.Int64
		}

		downstreams = append(downstreams, &downstream)
	}

	return downstreams, nil
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
