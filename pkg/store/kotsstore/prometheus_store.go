package kotsstore

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/rqlite/gorqlite"
)

func (s *KOTSStore) GetPrometheusAddress() (string, error) {
	db := persistence.MustGetDBSession()
	query := `select value from kotsadm_params where key = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{"PROMETHEUS_ADDRESS"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return "", nil
	}

	var value string
	if err := rows.Scan(&value); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return value, nil
}

func (s *KOTSStore) SetPrometheusAddress(address string) error {
	db := persistence.MustGetDBSession()
	query := `insert into kotsadm_params (key, value) values (?, ?) on conflict (key) do update set value = EXCLUDED.value`

	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{"PROMETHEUS_ADDRESS", address},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}
