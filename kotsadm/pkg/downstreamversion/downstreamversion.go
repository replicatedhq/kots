package downstreamversion

import (
	"database/sql"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstreamversion/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
)

// GetCurrentVersion returns the current version for an app downstream (sequence = current sequence)
func GetCurrentVersion(appID string, clusterID string) (*types.DownstreamVersion, error) {
	db := persistence.MustGetPGSession()
	query := "select current_sequence from app_downstream where app_id = $1 and cluster_id = $2"
	row := db.QueryRow(query, appID, clusterID)

	var currentSequence sql.NullInt64
	if err := row.Scan(&currentSequence); err != nil {
		return nil, errors.Wrap(err, "failed to scan sequence")
	}

	if !currentSequence.Valid {
		return nil, nil
	}
	sequence := currentSequence.Int64

	query = `select sequence, parent_sequence from app_downstream_version where app_id = $1 and cluster_id = $2 and sequence = $3`
	row = db.QueryRow(query, appID, clusterID, sequence)

	version := types.DownstreamVersion{}
	if err := row.Scan(&version.Sequence, &version.ParentSequence); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to scan version")
	}

	return &version, nil
}

// GetPendingVersions lists pending versions for an app downstream (sequence > current sequence)
func GetPendingVersions(appID string, clusterID string) ([]types.DownstreamVersion, error) {
	db := persistence.MustGetPGSession()
	query := "select current_sequence from app_downstream where app_id = $1 and cluster_id = $2"
	row := db.QueryRow(query, appID, clusterID)

	var currentSequence sql.NullInt64
	if err := row.Scan(&currentSequence); err != nil {
		return nil, errors.Wrap(err, "failed to scan sequence")
	}

	// If there is not a current_sequence, then all versions are future versions
	var sequence int64
	if currentSequence.Valid {
		sequence = currentSequence.Int64
	} else {
		sequence = -1
	}

	query = `select sequence, parent_sequence from app_downstream_version where app_id = $1 and cluster_id = $2 and sequence > $3 order by sequence desc`
	rows, err := db.Query(query, appID, clusterID, sequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query db")
	}

	versions := []types.DownstreamVersion{}
	for rows.Next() {
		v := types.DownstreamVersion{}
		if err := rows.Scan(&v.Sequence, &v.ParentSequence); err != nil {
			return nil, errors.Wrap(err, "failed to scan version")
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// SetVersionStatusReady sets the status for the downstream version with the given sequence and app id to "pending"
func SetVersionStatusReady(appID string, sequence int64) error {
	return setVersionStatus(appID, sequence, "pending")
}

// SetVersionStatusPendingPreflight sets the status for the downstream version with the given sequence and app id to "pending_preflight"
func SetVersionStatusPendingPreflight(appID string, sequence int64) error {
	return setVersionStatus(appID, sequence, "pending_preflight")
}

func setVersionStatus(appID string, sequence int64, status string) error {
	db := persistence.MustGetPGSession()
	query := `update app_downstream_version set status = $3 where app_id = $1 and sequence = $2`
	_, err := db.Exec(query, appID, sequence, status)
	if err != nil {
		return errors.Wrap(err, "failed to set downstream version status")
	}
	return nil
}

// GetVersionStatus gets the status for the downstream version with the given sequence and app id
func GetVersionStatus(appID string, sequence int64) (string, error) {
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
