package kotsstore

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/rqlite/gorqlite"
)

func (s *KOTSStore) GetEmbeddedClusterAuthToken() (string, error) {
	db := persistence.MustGetDBSession()
	query := `select value from kotsadm_params where key = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{"embedded.cluster.auth.token"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return "", ErrNotFound
	}

	var token string
	if err := rows.Scan(&token); err != nil {
		return "", errors.Wrap(err, "scan embedded cluster auth token")
	}

	return token, nil
}

func (s *KOTSStore) SetEmbeddedClusterAuthToken(token string) error {
	db := persistence.MustGetDBSession()

	query := `delete from kotsadm_params where key = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{"embedded.cluster.auth.token"},
	})
	if err != nil {
		return fmt.Errorf("delete embedded cluster auth token: %v: %v", err, wr.Err)
	}

	query = `insert into kotsadm_params (key, value) values (?, ?)`
	wr, err = db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{"embedded.cluster.auth.token", token},
	})
	if err != nil {
		return fmt.Errorf("insert embedded cluster auth token: %v: %v", err, wr.Err)
	}

	return nil
}
