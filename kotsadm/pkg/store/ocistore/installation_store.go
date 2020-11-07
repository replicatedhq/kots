package ocistore

import (
	"database/sql"

	"github.com/pkg/errors"
	installationtypes "github.com/replicatedhq/kots/kotsadm/pkg/online/types"
)

func (s OCIStore) GetPendingInstallationStatus() (*installationtypes.InstallStatus, error) {
	query := `SELECT install_state from app ORDER BY created_at DESC LIMIT 1`
	row := s.connection.DB.QueryRow(query)

	var installState sql.NullString
	if err := row.Scan(&installState); err != nil {
		if err == sql.ErrNoRows {
			return &installationtypes.InstallStatus{
				InstallStatus:  "not_installed",
				CurrentMessage: "",
			}, nil
		}
		return nil, errors.Wrap(err, "failed to scan")
	}

	_, message, err := s.GetTaskStatus("online-install")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get task status")
	}

	status := &installationtypes.InstallStatus{
		InstallStatus:  installState.String,
		CurrentMessage: message,
	}

	return status, nil
}
