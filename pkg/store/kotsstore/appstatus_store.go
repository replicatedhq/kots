package kotsstore

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/rqlite/gorqlite"
)

func (s *KOTSStore) GetAppStatus(appID string) (*appstatetypes.AppStatus, error) {
	db := persistence.MustGetDBSession()
	query := `select resource_states, updated_at, sequence from app_status where app_id = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	if !rows.Next() {
		return &appstatetypes.AppStatus{
			AppID:          appID,
			UpdatedAt:      time.Time{},
			ResourceStates: appstatetypes.ResourceStates{},
			State:          appstatetypes.StateMissing,
			Sequence:       0,
		}, nil
	}

	var updatedAt gorqlite.NullTime
	var resourceStatesStr gorqlite.NullString
	var sequence gorqlite.NullInt64

	if err := rows.Scan(&resourceStatesStr, &updatedAt, &sequence); err != nil {
		return nil, errors.Wrap(err, "failed to scan")
	}

	appStatus := appstatetypes.AppStatus{
		AppID:    appID,
		Sequence: sequence.Int64,
	}

	if updatedAt.Valid {
		appStatus.UpdatedAt = updatedAt.Time
	}

	if resourceStatesStr.Valid {
		var resourceStates appstatetypes.ResourceStates
		if err := json.Unmarshal([]byte(resourceStatesStr.String), &resourceStates); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal resource states")
		}
		appStatus.ResourceStates = resourceStates
	}

	appStatus.State = appstatetypes.GetState(appStatus.ResourceStates)

	return &appStatus, nil
}

func (s *KOTSStore) SetAppStatus(appID string, resourceStates appstatetypes.ResourceStates, updatedAt time.Time, sequence int64) error {
	marshalledResourceStates, err := json.Marshal(resourceStates)
	if err != nil {
		return errors.Wrap(err, "failed to json marshal resource states")
	}

	db := persistence.MustGetDBSession()
	query := `
	insert into app_status (app_id, resource_states, updated_at, sequence)
	values (?, ?, ?, ?)
	on conflict (app_id) do update set
	  resource_states = EXCLUDED.resource_states,
	  updated_at = EXCLUDED.updated_at,
	  sequence = EXCLUDED.sequence`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, string(marshalledResourceStates), updatedAt.Unix(), sequence},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}
