package kotsstore

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	appstatustypes "github.com/replicatedhq/kots/pkg/api/appstatus/types"
	"github.com/replicatedhq/kots/pkg/appstatus"
	"github.com/replicatedhq/kots/pkg/persistence"
)

func (s *KOTSStore) GetAppStatus(appID string) (*appstatustypes.AppStatus, error) {
	db := persistence.MustGetDBSession()
	query := `select resource_states, updated_at, sequence from app_status where app_id = $1`
	row := db.QueryRow(query, appID)

	var updatedAt sql.NullTime
	var resourceStatesStr sql.NullString
	var sequence sql.NullInt64

	if err := row.Scan(&resourceStatesStr, &updatedAt, &sequence); err != nil {
		if err == sql.ErrNoRows {
			return &appstatustypes.AppStatus{
				AppID:          appID,
				UpdatedAt:      time.Time{},
				ResourceStates: []appstatustypes.ResourceState{},
				State:          appstatustypes.StateMissing,
				Sequence:       0,
			}, nil
		}
		return nil, errors.Wrap(err, "failed to scan")
	}

	appStatus := appstatustypes.AppStatus{
		AppID:    appID,
		Sequence: sequence.Int64,
	}

	if updatedAt.Valid {
		appStatus.UpdatedAt = updatedAt.Time
	}

	if resourceStatesStr.Valid {
		var resourceStates []appstatustypes.ResourceState
		if err := json.Unmarshal([]byte(resourceStatesStr.String), &resourceStates); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal resource states")
		}
		appStatus.ResourceStates = resourceStates
	}

	appStatus.State = appstatus.GetState(appStatus.ResourceStates)

	return &appStatus, nil
}

func (s *KOTSStore) SetAppStatus(appID string, resourceStates []appstatustypes.ResourceState, updatedAt time.Time, sequence int64) error {
	marshalledResourceStates, err := json.Marshal(resourceStates)
	if err != nil {
		return errors.Wrap(err, "failed to json marshal resource states")
	}

	db := persistence.MustGetDBSession()
	query := `insert into app_status (app_id, resource_states, updated_at, sequence) values ($1, $2, $3, $4) on conflict (app_id) do update set resource_states = $2, updated_at = $3, sequence = $4`
	_, err = db.Exec(query, appID, marshalledResourceStates, updatedAt, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}
