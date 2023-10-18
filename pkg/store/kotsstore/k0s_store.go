package kotsstore

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/rqlite/gorqlite"
)

func (s *KOTSStore) SetK0sInstallCommandRoles(roles []string) (string, error) {
	db := persistence.MustGetDBSession()

	installID := uuid.New().String()

	query := `delete from k0s_tokens where token = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{installID},
	})
	if err != nil {
		return "", fmt.Errorf("delete k0s join token: %v: %v", err, wr.Err)
	}

	jsonRoles, err := json.Marshal(roles)
	if err != nil {
		return "", fmt.Errorf("failed to marshal roles: %w", err)
	}

	query = `insert into k0s_tokens (token, roles) values (?, ?)`
	wr, err = db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{installID, string(jsonRoles)},
	})
	if err != nil {
		return "", fmt.Errorf("insert k0s join token: %v: %v", err, wr.Err)
	}

	return installID, nil
}

func (s *KOTSStore) GetK0sInstallCommandRoles(token string) ([]string, error) {
	db := persistence.MustGetDBSession()
	query := `select roles from k0s_tokens where token = ?`
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
