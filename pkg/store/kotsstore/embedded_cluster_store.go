package kotsstore

import (
	"encoding/json"
	"fmt"
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
		return "", fmt.Errorf("delete embedded_cluster join token: %v: %v", err, wr.Err)
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
		return "", fmt.Errorf("insert embedded_cluster join token: %v: %v", err, wr.Err)
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
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
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
