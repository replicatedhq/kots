package s3pg

import (
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"go.uber.org/zap"
)

func (c S3PGStore) DeletePendingScheduledSnapshots(appID string) error {
	logger.Debug("Deleting pending scheduled snapshots",
		zap.String("appID", appID))
	db := persistence.MustGetPGSession()
	query := `DELETE FROM scheduled_snapshots WHERE app_id = $1 AND backup_name IS NULL`
	_, err := db.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to exec db query")
	}

	return nil
}

func (c S3PGStore) CreateScheduledSnapshot(id string, appID string, timestamp time.Time) error {
	logger.Debug("Creating scheduled snapshot",
		zap.String("appID", appID))
	db := persistence.MustGetPGSession()
	query := `
      INSERT INTO scheduled_snapshots (
        id,
        app_id,
        scheduled_timestamp
      ) VALUES (
        $1,
        $2,
        $3
      )
    `
	_, err := db.Exec(query, id, appID, timestamp)
	if err != nil {
		return errors.Wrap(err, "Failed to exec db query")
	}

	return nil
}
