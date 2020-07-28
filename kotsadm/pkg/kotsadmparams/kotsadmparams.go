package kotsadmparams

import (
	"database/sql"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
)

func Get(name string) (string, error) {
	db := persistence.MustGetPGSession()
	query := `select value from kotsadm_params where key = $1`
	row := db.QueryRow(query, name)

	var value string
	if err := row.Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", errors.Wrap(err, "failed to scan")
	}

	return value, nil
}

func Set(name string, value string) error {
	db := persistence.MustGetPGSession()
	query := `insert into kotsadm_params (key, value) values ($1, $2) on conflict (key) do update set value = $2`

	_, err := db.Exec(query, name, value)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}
