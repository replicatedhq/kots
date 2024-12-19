package kotsstore

import (
	"fmt"
	"time"

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

func (s *KOTSStore) UpsertPlan(p *types.Plan) error {
	db := persistence.MustGetDBSession()

	query := `
		INSERT INTO plan (app_id, version_label, created_at, updated_at, status, plan)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (app_id, version_label)
		DO UPDATE SET
			updated_at = excluded.updated_at,
			status = excluded.status,
			plan = excluded.plan
	`

	marshalled, err := yaml.Marshal(p)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}

	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query: query,
		Arguments: []interface{}{
			p.AppID,
			p.VersionLabel,
			time.Now().Unix(),
			time.Now().Unix(),
			p.GetStatus(),
			string(marshalled),
		},
	})
	if err != nil {
		return fmt.Errorf("write: %v: %v", err, wr.Err)
	}

	return err
}

func (s *KOTSStore) GetCurrentPlan(appID string) (*types.Plan, *time.Time, error) {
	db := persistence.MustGetDBSession()
	query := `SELECT plan, updated_at FROM plan WHERE app_id = ? ORDER BY updated_at DESC LIMIT 1`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return nil, nil, nil
	}

	var marshalled string
	var updatedAt time.Time
	if err := rows.Scan(&marshalled, &updatedAt); err != nil {
		return nil, nil, errors.Wrap(err, "scan")
	}

	var plan *types.Plan
	if err := yaml.Unmarshal([]byte(marshalled), &plan); err != nil {
		return nil, nil, errors.Wrap(err, "unmarshal")
	}

	return plan, &updatedAt, nil
}
