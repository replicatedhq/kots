package kotsstore

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/airgap/types"
	airgaptypes "github.com/replicatedhq/kots/pkg/airgap/types"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/tasks"
	"github.com/rqlite/gorqlite"
)

func (s *KOTSStore) GetPendingAirgapUploadApp() (*airgaptypes.PendingApp, error) {
	db := persistence.MustGetDBSession()
	query := `select id from app where install_state in ('airgap_upload_pending', 'airgap_upload_in_progress', 'airgap_upload_error') order by created_at desc limit 1`

	rows, err := db.QueryOne(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending app: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return nil, ErrNotFound
	}

	id := ""
	if err := rows.Scan(&id); err != nil {
		return nil, errors.Wrap(err, "failed to scan pending app id")
	}

	query = `select id, slug, name, license, channel_id from app where id = ?`
	rows, err = db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{id},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query app: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return nil, ErrNotFound
	}

	pendingApp := airgaptypes.PendingApp{}
	if err := rows.Scan(&pendingApp.ID, &pendingApp.Slug, &pendingApp.Name, &pendingApp.LicenseData, &pendingApp.ChannelID); err != nil {
		return nil, errors.Wrap(err, "failed to scan pending app")
	}

	return &pendingApp, nil
}

func (s *KOTSStore) GetAirgapInstallStatus(appID string) (*airgaptypes.InstallStatus, error) {
	db := persistence.MustGetDBSession()
	query := `SELECT slug, install_state FROM app WHERE id = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	if !rows.Next() {
		return &types.InstallStatus{
			InstallStatus:  "not_installed",
			CurrentMessage: "",
		}, nil
	}

	var slug string
	var installState gorqlite.NullString
	if err := rows.Scan(&slug, &installState); err != nil {
		return nil, errors.Wrap(err, "failed to scan")
	}

	_, message, err := tasks.GetTaskStatus(fmt.Sprintf("airgap-install-slug-%s", slug))
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

	query := `update app set install_state = 'airgap_upload_in_progress' where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID},
	})
	if err != nil {
		return fmt.Errorf("failed to set update airgap install status: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) SetAppIsAirgap(appID string, isAirgap bool) error {
	db := persistence.MustGetDBSession()

	query := `update app set is_airgap = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{isAirgap, appID},
	})
	if err != nil {
		return fmt.Errorf("failed to set app airgap flag: %v: %v", err, wr.Err)
	}

	return nil
}
