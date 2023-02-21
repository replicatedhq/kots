package kotsstore

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	snapshottypes "github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/rqlite/gorqlite"
	"go.uber.org/zap"
)

func (s *KOTSStore) ListPendingScheduledSnapshots(appID string) ([]snapshottypes.ScheduledSnapshot, error) {
	logger.Debug("Listing pending scheduled snapshots",
		zap.String("appID", appID))

	db := persistence.MustGetDBSession()
	query := `SELECT id, app_id, scheduled_timestamp FROM scheduled_snapshots WHERE app_id = ? AND backup_name IS NULL;`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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

func (s *KOTSStore) UpdateScheduledSnapshot(snapshotID string, backupName string) error {
	logger.Debug("Updating scheduled snapshot",
		zap.String("ID", snapshotID))

	db := persistence.MustGetDBSession()
	query := `UPDATE scheduled_snapshots SET backup_name = ? WHERE id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{backupName, snapshotID},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}
	return nil
}

func (s *KOTSStore) DeletePendingScheduledSnapshots(appID string) error {
	logger.Debug("Deleting pending scheduled snapshots",
		zap.String("appID", appID))

	db := persistence.MustGetDBSession()
	query := `DELETE FROM scheduled_snapshots WHERE app_id = ? AND backup_name IS NULL`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) CreateScheduledSnapshot(id string, appID string, timestamp time.Time) error {
	logger.Debug("Creating scheduled snapshot",
		zap.String("appID", appID))

	db := persistence.MustGetDBSession()
	query := `
		INSERT INTO scheduled_snapshots (
			id,
			app_id,
			scheduled_timestamp
		) VALUES (
			?,
			?,
			?
		)
	`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{id, appID, timestamp.Unix()},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) ListPendingScheduledInstanceSnapshots(clusterID string) ([]snapshottypes.ScheduledInstanceSnapshot, error) {
	logger.Debug("Listing pending scheduled instance snapshots",
		zap.String("clusterID", clusterID))

	db := persistence.MustGetDBSession()
	query := `SELECT id, cluster_id, scheduled_timestamp FROM scheduled_instance_snapshots WHERE cluster_id = ? AND backup_name IS NULL;`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{clusterID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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

func (s *KOTSStore) UpdateScheduledInstanceSnapshot(snapshotID string, backupName string) error {
	logger.Debug("Updating scheduled instance snapshot",
		zap.String("ID", snapshotID))

	db := persistence.MustGetDBSession()
	query := `UPDATE scheduled_instance_snapshots SET backup_name = ? WHERE id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{backupName, snapshotID},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}
	return nil
}

func (s *KOTSStore) DeletePendingScheduledInstanceSnapshots(clusterID string) error {
	logger.Debug("Deleting pending scheduled instance snapshots",
		zap.String("clusterID", clusterID))

	db := persistence.MustGetDBSession()
	query := `DELETE FROM scheduled_instance_snapshots WHERE cluster_id = ? AND backup_name IS NULL`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{clusterID},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) CreateScheduledInstanceSnapshot(id string, clusterID string, timestamp time.Time) error {
	logger.Debug("Creating scheduled instance snapshot",
		zap.String("clusterID", clusterID))

	db := persistence.MustGetDBSession()
	query := `
		INSERT INTO scheduled_instance_snapshots (
			id,
			cluster_id,
			scheduled_timestamp
		) VALUES (
			?,
			?,
			?
		)
	`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{id, clusterID, timestamp.Unix()},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}
