package app

import (
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/persistence"
)

// LastUpdateAtTime sets the time that the client last checked for an update to now
func LastUpdateAtTime(appID string) error {
	db := persistence.MustGetDBSession()
	query := `update app set last_update_check_at = $1 where id = $2`
	_, err := db.Exec(query, time.Now(), appID)
	if err != nil {
		return errors.Wrap(err, "failed to update last_update_check_at")
	}

	return nil
}

func InitiateRestore(snapshotName string, appID string) error {
	db := persistence.MustGetDBSession()
	query := `update app set restore_in_progress_name = $1 where id = $2`
	_, err := db.Exec(query, snapshotName, appID)
	if err != nil {
		return errors.Wrap(err, "failed to update restore_in_progress_name")
	}

	return nil
}

func ResetRestore(appID string) error {
	db := persistence.MustGetDBSession()
	query := `update app set restore_in_progress_name = NULL, restore_undeploy_status = '' where id = $1`
	_, err := db.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}

func SetRestoreUndeployStatus(appID string, undeployStatus types.UndeployStatus) error {
	db := persistence.MustGetDBSession()
	query := `update app set restore_undeploy_status = $1 where id = $2`
	_, err := db.Exec(query, undeployStatus, appID)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}
