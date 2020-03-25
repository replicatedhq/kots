package downstream

import (
	"database/sql"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/downstream/types"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
)

func ListDownstreamsForApp(appID string) ([]*types.Downstream, error) {
	db := persistence.MustGetPGSession()
	query := `select cluster_id, downstream_name, current_sequence from app_downstream where app_id = $1`
	rows, err := db.Query(query, appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get downstreams")
	}

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
