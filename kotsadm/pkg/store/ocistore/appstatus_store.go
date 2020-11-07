package ocistore

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/ocidb/ocidb/pkg/ocidb"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/appstatus"
	appstatustypes "github.com/replicatedhq/kots/kotsadm/pkg/appstatus/types"
)

func (s OCIStore) GetAppStatus(appID string) (*appstatustypes.AppStatus, error) {
	query := `select resource_states, updated_at from app_status where app_id = $1`
	row := s.connection.DB.QueryRow(query, appID)

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

	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return nil, errors.Wrap(err, "failed to commit")
	}

	return &appStatus, nil
}
