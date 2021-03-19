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

func (s KOTSStore) GetAppStatus(appID string) (*appstatustypes.AppStatus, error) {
	db := persistence.MustGetPGSession()
	query := `select resource_states, updated_at from app_status where app_id = $1`
	row := db.QueryRow(query, appID)

	var updatedAt sql.NullTime
	var resourceStatesStr sql.NullString

	if err := row.Scan(&resourceStatesStr, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return &appstatustypes.AppStatus{
				AppID:          appID,
				UpdatedAt:      time.Time{},
				ResourceStates: []appstatustypes.ResourceState{},
				State:          appstatustypes.StateMissing,
			}, nil
		}
		return nil, errors.Wrap(err, "failed to scan")
	}

	appStatus := appstatustypes.AppStatus{
		AppID: appID,
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

func (s KOTSStore) SetAppStatus(appID string, resourceStates []appstatustypes.ResourceState, updatedAt time.Time) error {
	marshalledResourceStates, err := json.Marshal(resourceStates)
	if err != nil {
		return errors.Wrap(err, "failed to json marshal resource states")
	}

	db := persistence.MustGetPGSession()
	query := `insert into app_status (app_id, resource_states, updated_at) values ($1, $2, $3) on conflict (app_id) do update set resource_states = $2, updated_at = $3`
	_, err = db.Exec(query, appID, marshalledResourceStates, updatedAt)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}
