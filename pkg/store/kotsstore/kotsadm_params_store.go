package kotsstore

import (
	"database/sql"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
)

// IsKotsadmIDGenerated retrieves the id of kotsadm if the pod is already
func (s *KOTSStore) IsKotsadmIDGenerated() (bool, error) {
	db := persistence.MustGetDBSession()
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

// SetIsKotsadmIDGenerated sets the status to true if the pod is starting for the first time
func (s *KOTSStore) SetIsKotsadmIDGenerated() error {
	db := persistence.MustGetDBSession()

	query := `insert into kotsadm_params (key, value) values ($1, $2) on conflict (key) do update set value = $2`
	_, err := db.Exec(query, "IS_KOTSADM_ID_GENERATED", true)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}
