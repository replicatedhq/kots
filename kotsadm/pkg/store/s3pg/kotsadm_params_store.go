package s3pg

import (
	"database/sql"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
)

// IsKotsadmIDGenerated retrieves the id of kotsadm if the pod is already
func (s S3PGStore) IsKotsadmIDGenerated() (bool, error) {
	db := persistence.MustGetPGSession()
	query := `select value from kotsadm_params where key = 'IS_KOTSADM_ID_GENERATED'`
	row := db.QueryRow(query)

	var value string
	if err := row.Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to scan")
	}
	return true, nil
}

// SetKotsAdmEventStatus sets the status to true if the pod is starting for the first time
func (s S3PGStore) SetKotsAdmEventStatus() error {
	db := persistence.MustGetPGSession()

	//TODO double check if this insert query is correct - when $2 will conflict?
	query := `insert into kotsadm_params (key, value) values ($1, $2) on conflict (key) do update set value = $2`

	_, err := db.Exec(query, "IS_KOTSADM_ID_GENERATED", true)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}
