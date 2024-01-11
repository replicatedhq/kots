package kotsstore

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rqlite/gorqlite"

	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/rand"
)

func (s *KOTSStore) SetEmbeddedClusterInstallCommandRoles(roles []string) (string, error) {
	db := persistence.MustGetDBSession()

	installID := rand.StringWithCharset(24, rand.LOWER_CASE+rand.UPPER_CASE)

	query := `delete from embedded_cluster_tokens where token = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{installID},
	})
	if err != nil {
		return "", fmt.Errorf("delete embedded_cluster join token: %w: %v", err, wr.Err)
	}

	jsonRoles, err := json.Marshal(roles)
	if err != nil {
		return "", fmt.Errorf("failed to marshal roles: %w", err)
	}

	query = `insert into embedded_cluster_tokens (token, roles) values (?, ?)`
	wr, err = db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{installID, string(jsonRoles)},
	})
	if err != nil {
		return "", fmt.Errorf("insert embedded_cluster join token: %w: %v", err, wr.Err)
	}

	return installID, nil
}

func (s *KOTSStore) GetEmbeddedClusterInstallCommandRoles(token string) ([]string, error) {
	db := persistence.MustGetDBSession()
	query := `select roles from embedded_cluster_tokens where token = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{token},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w: %v", err, rows.Err)
	}
	if !rows.Next() {
		return nil, ErrNotFound
	}

	rolesStr := ""
	if err = rows.Scan(&rolesStr); err != nil {
		return nil, fmt.Errorf("failed to scan roles: %w", err)
	}

	rolesArr := []string{}
	err = json.Unmarshal([]byte(rolesStr), &rolesArr)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal roles: %w", err)
	}

	return rolesArr, nil
}

func (s *KOTSStore) SetEmbeddedClusterState(state string) error {
	db := persistence.MustGetDBSession()
	query := `
insert into embedded_cluster_status (updated_at, status)
values (?, ?)
on conflict (updated_at) do update set
	  status = EXCLUDED.status`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{time.Now().Unix(), state},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %w: %v", err, wr.Err)
	}
	return nil
}

func (s *KOTSStore) GetEmbeddedClusterState() (string, error) {
	db := persistence.MustGetDBSession()
	query := `select status from embedded_cluster_status ORDER BY updated_at DESC LIMIT 1`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query: %w: %v", err, rows.Err)
	}
	if !rows.Next() {
		return "", nil
	}
	var state gorqlite.NullString
	if err := rows.Scan(&state); err != nil {
		return "", fmt.Errorf("failed to scan: %w", err)
	}
	return state.String, nil
}
