package ocistore

import (
	"context"
	"database/sql"

	"github.com/ocidb/ocidb/pkg/ocidb"
	"github.com/pkg/errors"
	airgaptypes "github.com/replicatedhq/kots/kotsadm/pkg/airgap/types"
)

func (s OCIStore) GetPendingAirgapUploadApp() (*airgaptypes.PendingApp, error) {
	query := `select id from app where install_state in ('airgap_upload_pending', 'airgap_upload_in_progress', 'airgap_upload_error') order by created_at desc limit 1`
	row := s.connection.DB.QueryRow(query)

	id := ""
	if err := row.Scan(&id); err != nil {
		return nil, errors.Wrap(err, "failed to scan pending app id")
	}

	query = `select id, slug, name, license from app where id = $1`
	row = s.connection.DB.QueryRow(query, id)

	pendingApp := airgaptypes.PendingApp{}
	if err := row.Scan(&pendingApp.ID, &pendingApp.Slug, &pendingApp.Name, &pendingApp.LicenseData); err != nil {
		return nil, errors.Wrap(err, "failed to scan pending app")
	}

	return &pendingApp, nil
}

func (s OCIStore) GetAirgapInstallStatus() (*airgaptypes.InstallStatus, error) {
	query := `SELECT install_state from app ORDER BY created_at DESC LIMIT 1`
	row := s.connection.DB.QueryRow(query)

	var installState sql.NullString
	if err := row.Scan(&installState); err != nil {
		if err == sql.ErrNoRows {
			return &airgaptypes.InstallStatus{
				InstallStatus:  "not_installed",
				CurrentMessage: "",
			}, nil
		}
		return nil, errors.Wrap(err, "failed to scan")
	}

	_, message, err := s.GetTaskStatus("airgap-install")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get task status")
	}

	status := &airgaptypes.InstallStatus{
		InstallStatus:  installState.String,
		CurrentMessage: message,
	}

	return status, nil
}

func (s OCIStore) ResetAirgapInstallInProgress(appID string) error {
	query := `update app set install_state = 'airgap_upload_in_progress' where id = $1`
	_, err := s.connection.DB.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to set update airgap install status")
	}
	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}

func (s OCIStore) SetAppIsAirgap(appID string, isAirgap bool) error {
	query := `update app set is_airgap=$1 where id = $2`
	_, err := s.connection.DB.Exec(query, isAirgap, appID)
	if err != nil {
		return errors.Wrap(err, "failed to set app airgap flag")
	}
	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}
