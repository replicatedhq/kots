package kotsstore

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/airgap/types"
	airgaptypes "github.com/replicatedhq/kots/pkg/airgap/types"
	"github.com/replicatedhq/kots/pkg/persistence"
)

func (s *KOTSStore) GetPendingAirgapUploadApp() (*airgaptypes.PendingApp, error) {
	db := persistence.MustGetDBSession()
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

func (s *KOTSStore) GetAirgapInstallStatus(appID string) (*airgaptypes.InstallStatus, error) {
	db := persistence.MustGetDBSession()
	query := `SELECT slug, install_state FROM app WHERE id = $1`
	row := db.QueryRow(query, appID)

	var slug string
	var installState sql.NullString
	if err := row.Scan(&slug, &installState); err != nil {
		if err == sql.ErrNoRows {
			return &types.InstallStatus{
				InstallStatus:  "not_installed",
				CurrentMessage: "",
			}, nil
		}
		return nil, errors.Wrap(err, "failed to scan")
	}

	_, message, err := s.GetTaskStatus(fmt.Sprintf("airgap-install-slug-%s", slug))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get task status")
	}

	status := &airgaptypes.InstallStatus{
		InstallStatus:  installState.String,
		CurrentMessage: message,
	}

	return status, nil
}

func (s *KOTSStore) ResetAirgapInstallInProgress(appID string) error {
	db := persistence.MustGetDBSession()

	query := `update app set install_state = 'airgap_upload_in_progress' where id = $1`
	_, err := db.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to set update airgap install status")
	}

	return nil
}

func (s *KOTSStore) SetAppIsAirgap(appID string, isAirgap bool) error {
	db := persistence.MustGetDBSession()

	query := `update app set is_airgap=$1 where id = $2`
	_, err := db.Exec(query, isAirgap, appID)
	if err != nil {
		return errors.Wrap(err, "failed to set app airgap flag")
	}

	return nil
}
