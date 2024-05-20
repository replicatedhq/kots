package kotsstore

import (
	"fmt"

	"github.com/pkg/errors"
	installationtypes "github.com/replicatedhq/kots/pkg/online/types"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/tasks"
	"github.com/rqlite/gorqlite"
)

func (s *KOTSStore) GetPendingInstallationStatus() (*installationtypes.InstallStatus, error) {
	db := persistence.MustGetDBSession()
	query := `SELECT install_state from app ORDER BY created_at DESC LIMIT 1`
	rows, err := db.QueryOne(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	if !rows.Next() {
		return &installationtypes.InstallStatus{
			InstallStatus:  "not_installed",
			CurrentMessage: "",
		}, nil
	}

	var installState gorqlite.NullString
	if err := rows.Scan(&installState); err != nil {
		return nil, errors.Wrap(err, "failed to scan")
	}

	_, message, err := tasks.GetTaskStatus("online-install")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get task status")
	}

	status := &installationtypes.InstallStatus{
		InstallStatus:  installState.String,
		CurrentMessage: message,
	}

	return status, nil
}
