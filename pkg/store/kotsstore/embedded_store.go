package kotsstore

import (
	"database/sql"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
)

func (s *KOTSStore) GetEmbeddedClusterAuthToken() (string, error) {
	pg := persistence.MustGetDBSession()
	query := `select value from kotsadm_params where key = $1`
	row := pg.QueryRow(query, "embedded.cluster.auth.token")

	var token string
	if err := row.Scan(&token); err != nil {
		if err == sql.ErrNoRows {
			return "", ErrNotFound
		}

		return "", errors.Wrap(err, "scan embedded cluster auth token")
	}

	return token, nil
}

func (s *KOTSStore) SetEmbeddedClusterAuthToken(token string) error {
	pg := persistence.MustGetDBSession()

	query := `delete from kotsadm_params where key = $1`
	_, err := pg.Exec(query, "embedded.cluster.auth.token")
	if err != nil {
		return errors.Wrap(err, "delete embedded cluster auth token")
	}

	query = `insert into kotsadm_params (key, value) values ($1, $2)`
	_, err = pg.Exec(query, "embedded.cluster.auth.token", token)
	if err != nil {
		return errors.Wrap(err, "insert embedded cluster auth token")
	}

	return nil
}
