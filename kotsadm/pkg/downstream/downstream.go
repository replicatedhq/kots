package downstream

import (
	"database/sql"
	"encoding/base64"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
)

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

func GetParentSequence(appID string, clusterID string) (int64, error) {
	db := persistence.MustGetPGSession()
	query := `select current_sequence from app_downstream where app_id = $1 and cluster_id = $2`
	row := db.QueryRow(query, appID, clusterID)

	var currentSequence sql.NullInt64
	if err := row.Scan(&currentSequence); err != nil {
		return 0, errors.Wrap(err, "failed to scan")
	}

	if !currentSequence.Valid {
		return -1, nil
	}

	query = `select parent_sequence from app_downstream_version where app_id = $1 and cluster_id = $2 and sequence = $3`
	row = db.QueryRow(query, appID, clusterID, currentSequence.Int64)

	var parentSequence sql.NullInt64
	if err := row.Scan(&parentSequence); err != nil {
		return 0, errors.Wrap(err, "failed to scan")
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
			return &types.DownstreamOutput{
				DryrunStdout: "",
				DryrunStderr: "",
				ApplyStdout:  "",
				ApplyStderr:  "",
				RenderError:  "",
			}, nil
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
