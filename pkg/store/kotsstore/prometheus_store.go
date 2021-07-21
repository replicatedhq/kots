package kotsstore

import (
	"database/sql"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
)

func (s *KOTSStore) GetPrometheusAddress() (string, error) {
	db := persistence.MustGetDBSession()
	query := `select value from kotsadm_params where key = $1`
	row := db.QueryRow(query, "PROMETHEUS_ADDRESS")

	var value string
	if err := row.Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", errors.Wrap(err, "failed to scan")
	}

	return value, nil
}

func (s *KOTSStore) SetPrometheusAddress(address string) error {
	db := persistence.MustGetDBSession()
	query := `insert into kotsadm_params (key, value) values ($1, $2) on conflict (key) do update set value = $2`

	_, err := db.Exec(query, "PROMETHEUS_ADDRESS", address)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}
