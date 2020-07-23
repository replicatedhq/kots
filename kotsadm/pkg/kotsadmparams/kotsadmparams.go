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
