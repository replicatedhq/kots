package app

import (
	"fmt"
	"time"

	"github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/rqlite/gorqlite"
)

// SetLastUpdateAtTime sets the time that the client last checked for an update to now
func SetLastUpdateAtTime(appID string, t time.Time) error {
	db := persistence.MustGetDBSession()
	query := `update app set last_update_check_at = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{t.Unix(), appID},
	})
	if err != nil {
		return fmt.Errorf("failed to update last_update_check_at: %v: %v", err, wr.Err)
	}

	return nil
}

func InitiateRestore(snapshotName string, appID string) error {
	db := persistence.MustGetDBSession()
	query := `update app set restore_in_progress_name = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{snapshotName, appID},
	})
	if err != nil {
		return fmt.Errorf("failed to update restore_in_progress_name: %v: %v", err, wr.Err)
	}

	return nil
}

func ResetRestore(appID string) error {
	db := persistence.MustGetDBSession()
	query := `update app set restore_in_progress_name = NULL, restore_undeploy_status = '' where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func SetRestoreUndeployStatus(appID string, undeployStatus types.UndeployStatus) error {
	db := persistence.MustGetDBSession()
	query := `update app set restore_undeploy_status = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{undeployStatus, appID},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}
