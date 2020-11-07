package ocistore

import (
	"context"
	"database/sql"

	"github.com/ocidb/ocidb/pkg/ocidb"
	"github.com/pkg/errors"
)

func (s OCIStore) GetPrometheusAddress() (string, error) {
	query := `select value from kotsadm_params where key = $1`
	row := s.connection.DB.QueryRow(query, "PROMETHEUS_ADDRESS")

	var value string
	if err := row.Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", errors.Wrap(err, "failed to scan")
	}

	return value, nil
}

func (s OCIStore) SetPrometheusAddress(address string) error {
	query := `insert into kotsadm_params (key, value) values ($1, $2) on conflict (key) do update set value = $2`

	_, err := s.connection.DB.Exec(query, "PROMETHEUS_ADDRESS", address)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}
