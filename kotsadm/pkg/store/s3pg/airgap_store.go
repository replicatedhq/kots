package s3pg

import (
	"database/sql"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/airgap/types"
	airgaptypes "github.com/replicatedhq/kots/kotsadm/pkg/airgap/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
)

func (s S3PGStore) GetPendingAirgapUploadApp() (*airgaptypes.PendingApp, error) {
	db := persistence.MustGetPGSession()
	query := `select id from app where install_state in ('airgap_upload_pending', 'airgap_upload_in_progress', 'airgap_upload_error') order by created_at desc limit 1`
	row := db.QueryRow(query)

	id := ""
	if err := row.Scan(&id); err != nil {
		return nil, errors.Wrap(err, "failed to scan pending app id")
	}

	query = `select id, slug, name, license from app where id = $1`
	row = db.QueryRow(query, id)

	pendingApp := airgaptypes.PendingApp{}
	if err := row.Scan(&pendingApp.ID, &pendingApp.Slug, &pendingApp.Name, &pendingApp.LicenseData); err != nil {
		return nil, errors.Wrap(err, "failed to scan pending app")
	}

	return &pendingApp, nil
}

func (s S3PGStore) GetAirgapInstallStatus() (*airgaptypes.InstallStatus, error) {
	db := persistence.MustGetPGSession()
	query := `SELECT install_state from app ORDER BY created_at DESC LIMIT 1`
	row := db.QueryRow(query)

	var installState sql.NullString
	if err := row.Scan(&installState); err != nil {
		if err == sql.ErrNoRows {
			return &types.InstallStatus{
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

func (s S3PGStore) ResetAirgapInstallInProgress(appID string) error {
	db := persistence.MustGetPGSession()

	query := `update app set install_state = 'airgap_upload_in_progress' where id = $1`
	_, err := db.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to set update airgap install status")
	}

	return nil
}

func (s S3PGStore) SetAppIsAirgap(appID string, isAirgap bool) error {
	db := persistence.MustGetPGSession()

	query := `update app set is_airgap=$1 where id = $2`
	_, err := db.Exec(query, isAirgap, appID)
	if err != nil {
		return errors.Wrap(err, "failed to set app airgap flag")
	}

	return nil
}

func (s S3PGStore) SetAppInstallState(appID string, state string) error {
	db := persistence.MustGetPGSession()

	query := `update app set install_state = $2 where id = $1`
	_, err := db.Exec(query, appID, state)
	if err != nil {
		return errors.Wrap(err, "failed to update app install state")
	}

	return nil
}
