package s3pg

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
)

func (s S3PGStore) ResetPreflightResult(appID string, sequence int64) error {
	db := persistence.MustGetPGSession()
	query := `update app_downstream_version set preflight_result=null, preflight_result_created_at=null where app_id = $1 and parent_sequence = $2`
	_, err := db.Exec(query, appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}
