package s3pg

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	snapshottypes "github.com/replicatedhq/kots/kotsadm/pkg/snapshot/types"
	"go.uber.org/zap"
)

func (c S3PGStore) ListPendingScheduledSnapshots(appID string) ([]snapshottypes.ScheduledSnapshot, error) {
	logger.Debug("Listing pending scheduled snapshots",
		zap.String("appID", appID))

	db := persistence.MustGetPGSession()
	query := `SELECT id, app_id, scheduled_timestamp FROM scheduled_snapshots WHERE app_id = $1 AND backup_name IS NULL;`
	rows, err := db.Query(query, appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()

	scheduledSnapshots := []snapshottypes.ScheduledSnapshot{}
	for rows.Next() {
		s := snapshottypes.ScheduledSnapshot{}
		if err := rows.Scan(&s.ID, &s.AppID, &s.ScheduledTimestamp); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		scheduledSnapshots = append(scheduledSnapshots, s)
	}

	return scheduledSnapshots, nil
}

func (c S3PGStore) UpdateScheduledSnapshot(tx *sql.Tx, ID string, backupName string) error {
	query := `UPDATE scheduled_snapshots SET backup_name = $1 WHERE id = $2`
	_, err := tx.Exec(query, backupName, ID)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (c S3PGStore) LockScheduledSnapshot(tx *sql.Tx, ID string) (bool, error) {
	query := `SELECT * FROM scheduled_snapshots WHERE id = $1 FOR UPDATE SKIP LOCKED`
	rows, err := tx.Query(query, ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()

	for rows.Next() {
		return true, nil
	}

	return false, nil
}

func (c S3PGStore) DeletePendingScheduledSnapshots(appID string, tx *sql.Tx) error {
	logger.Debug("Deleting pending scheduled snapshots",
		zap.String("appID", appID))

	query := `DELETE FROM scheduled_snapshots WHERE app_id = $1 AND backup_name IS NULL`

	if tx != nil {
		_, err := tx.Exec(query, appID)
		if err != nil {
			return errors.Wrap(err, "failed to tx exec query")
		}
	} else {
		db := persistence.MustGetPGSession()
		_, err := db.Exec(query, appID)
		if err != nil {
			return errors.Wrap(err, "failed to db exec query")
		}
	}

	return nil
}

func (c S3PGStore) CreateScheduledSnapshot(id string, appID string, timestamp time.Time, tx *sql.Tx) error {
	logger.Debug("Creating scheduled snapshot",
		zap.String("appID", appID))

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

	if tx != nil {
		_, err := tx.Exec(query, id, appID, timestamp)
		if err != nil {
			return errors.Wrap(err, "Failed to tx exec query")
		}
	} else {
		db := persistence.MustGetPGSession()
		_, err := db.Exec(query, id, appID, timestamp)
		if err != nil {
			return errors.Wrap(err, "Failed to db exec query")
		}
	}

	return nil
}
