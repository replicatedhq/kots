package kotsstore

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	"github.com/replicatedhq/kots/pkg/persistence"
)

func (s *KOTSStore) GetAppStatus(appID string) (*appstatetypes.AppStatus, error) {
	db := persistence.MustGetDBSession()
	query := `select resource_states, updated_at, sequence from app_status where app_id = $1`
	row := db.QueryRow(query, appID)

	var updatedAt sql.NullTime
	var resourceStatesStr sql.NullString
	var sequence sql.NullInt64

	if err := row.Scan(&resourceStatesStr, &updatedAt, &sequence); err != nil {
		if err == sql.ErrNoRows {
			return &appstatetypes.AppStatus{
				AppID:          appID,
				UpdatedAt:      time.Time{},
				ResourceStates: appstatetypes.ResourceStates{},
				State:          appstatetypes.StateMissing,
				Sequence:       0,
			}, nil
		}
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
	values ($1, $2, $3, $4)
	on conflict (app_id) do update set
	  resource_states = EXCLUDED.resource_states,
	  updated_at = EXCLUDED.updated_at,
	  sequence = EXCLUDED.sequence`
	_, err = db.Exec(query, appID, marshalledResourceStates, updatedAt, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}
