package s3pg

import (
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

func (c S3PGStore) UpdateScheduledSnapshot(snapshotID string, backupName string) error {
	logger.Debug("Updating scheduled snapshot",
		zap.String("ID", snapshotID))

	db := persistence.MustGetPGSession()
	query := `UPDATE scheduled_snapshots SET backup_name = $1 WHERE id = $2`
	_, err := db.Exec(query, backupName, snapshotID)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (c S3PGStore) DeletePendingScheduledSnapshots(appID string) error {
	logger.Debug("Deleting pending scheduled snapshots",
		zap.String("appID", appID))

	db := persistence.MustGetPGSession()
	query := `DELETE FROM scheduled_snapshots WHERE app_id = $1 AND backup_name IS NULL`
	_, err := db.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to db exec query")
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
		return errors.Wrap(err, "Failed to db exec query")
	}

	return nil
}

func (c S3PGStore) ListPendingScheduledInstanceSnapshots(clusterID string) ([]snapshottypes.ScheduledInstanceSnapshot, error) {
	logger.Debug("Listing pending scheduled instance snapshots",
		zap.String("clusterID", clusterID))

	db := persistence.MustGetPGSession()
	query := `SELECT id, cluster_id, scheduled_timestamp FROM scheduled_instance_snapshots WHERE cluster_id = $1 AND backup_name IS NULL;`
	rows, err := db.Query(query, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()

	scheduledSnapshots := []snapshottypes.ScheduledInstanceSnapshot{}
	for rows.Next() {
		s := snapshottypes.ScheduledInstanceSnapshot{}
		if err := rows.Scan(&s.ID, &s.ClusterID, &s.ScheduledTimestamp); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		scheduledSnapshots = append(scheduledSnapshots, s)
	}

	return scheduledSnapshots, nil
}

func (c S3PGStore) UpdateScheduledInstanceSnapshot(snapshotID string, backupName string) error {
	logger.Debug("Updating scheduled instance snapshot",
		zap.String("ID", snapshotID))

	db := persistence.MustGetPGSession()
	query := `UPDATE scheduled_instance_snapshots SET backup_name = $1 WHERE id = $2`
	_, err := db.Exec(query, backupName, snapshotID)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (c S3PGStore) DeletePendingScheduledInstanceSnapshots(clusterID string) error {
	logger.Debug("Deleting pending scheduled instance snapshots",
		zap.String("clusterID", clusterID))

	db := persistence.MustGetPGSession()
	query := `DELETE FROM scheduled_instance_snapshots WHERE cluster_id = $1 AND backup_name IS NULL`
	_, err := db.Exec(query, clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to db exec query")
	}

	return nil
}

func (c S3PGStore) CreateScheduledInstanceSnapshot(id string, clusterID string, timestamp time.Time) error {
	logger.Debug("Creating scheduled instance snapshot",
		zap.String("clusterID", clusterID))

	db := persistence.MustGetPGSession()
	query := `
		INSERT INTO scheduled_instance_snapshots (
			id,
			cluster_id,
			scheduled_timestamp
		) VALUES (
			$1,
			$2,
			$3
		)
	`
	_, err := db.Exec(query, id, clusterID, timestamp)
	if err != nil {
		return errors.Wrap(err, "Failed to db exec query")
	}

	return nil
}
