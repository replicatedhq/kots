package kotsstore

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/rqlite/gorqlite"
	"gopkg.in/yaml.v3"
)

func (s *KOTSStore) GetPlan(appID, versionLabel string) (*types.Plan, error) {
	db := persistence.MustGetDBSession()
	query := `SELECT plan FROM plan WHERE app_id = ? AND version_label = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, versionLabel},
	})
	if err != nil {
		return nil, fmt.Errorf("query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return nil, ErrNotFound
	}

	var marshalled string
	if err := rows.Scan(&marshalled); err != nil {
		return nil, fmt.Errorf("scan: %v", err)
	}

	var plan *types.Plan
	if err := yaml.Unmarshal([]byte(marshalled), &plan); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	return plan, nil
}

func (s *KOTSStore) UpsertPlan(appID string, versionLabel string, plan *types.Plan) error {
	db := persistence.MustGetDBSession()

	query := `
		INSERT INTO plan (app_id, version_label, plan)
		VALUES (?, ?, ?)
		ON CONFLICT (app_id, version_label)
		DO UPDATE SET plan = excluded.plan
	`

	marshalled, err := yaml.Marshal(plan)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}

	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, versionLabel, string(marshalled)},
	})
	if err != nil {
		return fmt.Errorf("write: %v: %v", err, wr.Err)
	}

	return err
}
