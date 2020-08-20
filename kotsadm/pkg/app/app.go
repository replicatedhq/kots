package app

import (
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"go.uber.org/zap"
)

func GetLicenseDataFromDatabase(id string) (string, error) {
	logger.Debug("getting app license from database",
		zap.String("id", id))

	db := persistence.MustGetPGSession()
	query := `select license from app where id = $1`
	row := db.QueryRow(query, id)

	license := ""

	if err := row.Scan(&license); err != nil {
		return "", errors.Wrap(err, "failed to scan license")
	}

	return license, nil
}

// LastUpdateAtTime sets the time that the client last checked for an update to now
func LastUpdateAtTime(appID string) error {
	db := persistence.MustGetPGSession()
	query := `update app set last_update_check_at = $1 where id = $2`
	_, err := db.Exec(query, time.Now(), appID)
	if err != nil {
		return errors.Wrap(err, "failed to update last_update_check_at")
	}

	return nil
}

func InitiateRestore(snapshotName string, appID string) error {
	db := persistence.MustGetPGSession()
	query := `update app set restore_in_progress_name = $1 where id = $2`
	_, err := db.Exec(query, snapshotName, appID)
	if err != nil {
		return errors.Wrap(err, "failed to update restore_in_progress_name")
	}

	return nil
}
