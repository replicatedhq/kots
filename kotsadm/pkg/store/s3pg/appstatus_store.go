package s3pg

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/appstatus"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	appstatustypes "github.com/replicatedhq/kots/pkg/api/appstatus/types"
)

func (s S3PGStore) GetAppStatus(appID string) (*appstatustypes.AppStatus, error) {
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
