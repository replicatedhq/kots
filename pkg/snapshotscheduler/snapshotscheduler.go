package snapshotscheduler

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	snapshot "github.com/replicatedhq/kots/pkg/kotsadmsnapshot"
	snapshottypes "github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	"k8s.io/apimachinery/pkg/util/rand"

	cron "github.com/robfig/cron/v3"
)

func Start() error {
	logger.Debug("starting snapshot scheduler")

	startLoop(appScheduleLoop, 60)
	startLoop(instanceScheduleLoop, 60)

	return nil
}

func startLoop(fn func(), intervalInSeconds time.Duration) {
	go func() {
		for {
			fn()
			time.Sleep(time.Second * intervalInSeconds)
		}
	}()
}

func appScheduleLoop() {
	appsList, err := store.GetStore().ListInstalledApps()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list installed apps for scheduled snapshots"))
		return
	}

	for _, a := range appsList {
		if a.RestoreInProgressName != "" {
			continue
		}
		if err := handleApp(a); err != nil {
			logger.Error(errors.Wrapf(err, "failed to handle scheduled snapshots for app %s", a.ID))
		}
	}
}

func instanceScheduleLoop() {
	clusters, err := store.GetStore().ListClusters()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list clusters for scheduled instance snapshots"))
		return
	}

	for _, c := range clusters {
		if err := handleCluster(c); err != nil {
			logger.Error(errors.Wrapf(err, "failed to handle scheduled instance snapshots for cluster %s", c.ClusterID))
		}
	}
}

/* App Level Scheduled Snapshots */
func handleApp(a *apptypes.App) error {
	if a.SnapshotSchedule == "" {
		return nil
	}

	/*
	* This queue uses the scheduled_snapshots table to keep track of the next scheduled snapshot
	* for each app. Nothing else uses this table.
	*
	* For each app, list all pending snapshots. If scheduled snapshots are enabled there should be
	* exactly 1 pending snapshot for the app. (If the table has been manually edited and there are
	* 0 or 2+ pending snapshots this routine will fix it up so there's exactly 1 when it finishes.)
	*
	* Before taking a snapshot, first check that it's not scheduled for a time in the future, then
	* check that there is not already another snapshot in progress for the app. If both of those
	* checks pass, then create the Backup CR for velero, save the Backup name to the row to
	* mark that it has been handled, then schedule the next snapshot from the app's cron schedule
	* expression.
	 */

	pending, err := store.GetStore().ListPendingScheduledSnapshots(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to list pending scheduled snapshots")
	}

	if len(pending) == 0 {
		logger.Infof("No pending snapshots scheduled for app %s with schedule %s. Queueing one.", a.ID, a.SnapshotSchedule)
		queued, err := nextScheduledApplicationSnapshot(a.ID, a.SnapshotSchedule)
		if err != nil {
			return errors.Wrap(err, "failed to get next schedule")
		}
		if err := store.GetStore().CreateScheduledSnapshot(queued.ID, queued.AppID, queued.ScheduledTimestamp); err != nil {
			return errors.Wrap(err, "failed to create scheduled snapshot")
		}
		return nil
	}

	next := pending[0]
	if next.ScheduledTimestamp.After(time.Now()) {
		logger.Debugf("Not yet time to snapshot app %s", a.ID)
		return nil
	}

	hasUnfinished, err := snapshot.HasUnfinishedApplicationBackup(context.Background(), util.PodNamespace, a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to to check if app has unfinished backups")
	}
	if hasUnfinished {
		logger.Infof("Postponing scheduled application snapshot for app %s because one is in progress", a.ID)
		return nil
	}

	backup, err := snapshot.CreateApplicationBackup(context.Background(), a, true)
	if err != nil {
		return errors.Wrap(err, "failed to create backup")
	}

	if err := store.GetStore().UpdateScheduledSnapshot(next.ID, backup.ObjectMeta.Name); err != nil {
		return errors.Wrap(err, "failed to update scheduled snapshot")
	}
	logger.Infof("Created application backup %s from scheduled snapshot %s", backup.ObjectMeta.Name, next.ID)

	if len(pending) > 1 {
		err := store.GetStore().DeletePendingScheduledSnapshots(a.ID)
		if err != nil {
			return errors.Wrap(err, "failed to delete pending scheduled snapshots")
		}
	}

	queued, err := nextScheduledApplicationSnapshot(a.ID, a.SnapshotSchedule)
	if err != nil {
		return errors.Wrap(err, "failed to get next schedule")
	}

	if err := store.GetStore().CreateScheduledSnapshot(queued.ID, queued.AppID, queued.ScheduledTimestamp); err != nil {
		return errors.Wrap(err, "failed to create scheduled snapshot")
	}
	logger.Infof("Scheduled next application snapshot %s", queued.ID)

	return nil
}

/* Cluster/Instance Level Scheduled Snapshots */
func handleCluster(c *downstreamtypes.Downstream) error {
	if c.SnapshotSchedule == "" {
		return nil
	}

	/*
	* This queue uses the scheduled_instance_snapshots table to keep track of the next scheduled instance snapshot
	* for each cluster. Nothing else uses this table.
	*
	* For each cluster, list all pending snapshots. If scheduled instance snapshots are enabled there should be
	* exactly 1 pending snapshot for the cluster. (If the table has been manually edited and there are
	* 0 or 2+ pending snapshots this routine will fix it up so there's exactly 1 when it finishes.)
	*
	* Before taking a snapshot, first check that it's not scheduled for a time in the future, then
	* check that there is not already another snapshot in progress for the cluster. If both of those
	* checks pass, then create the Backup CR for velero, save the Backup name to the row to
	* mark that it has been handled, then schedule the next snapshot from the cluster's cron schedule
	* expression.
	 */

	pending, err := store.GetStore().ListPendingScheduledInstanceSnapshots(c.ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to list pending scheduled instance snapshots")
	}

	if len(pending) == 0 {
		logger.Infof("No pending instance snapshots scheduled for cluster %s with schedule %s. Queueing one.", c.ClusterID, c.SnapshotSchedule)
		queued, err := nextScheduledInstanceSnapshot(c.ClusterID, c.SnapshotSchedule)
		if err != nil {
			return errors.Wrap(err, "failed to get next schedule")
		}
		if err := store.GetStore().CreateScheduledInstanceSnapshot(queued.ID, queued.ClusterID, queued.ScheduledTimestamp); err != nil {
			return errors.Wrap(err, "failed to create scheduled instance snapshot")
		}
		return nil
	}

	next := pending[0]
	if next.ScheduledTimestamp.After(time.Now()) {
		logger.Debugf("Not yet time to snapshot instance/cluster %s", c.ClusterID)
		return nil
	}

	hasUnfinished, err := snapshot.HasUnfinishedInstanceBackup(context.Background(), util.PodNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to to check if cluster has unfinished backups")
	}
	if hasUnfinished {
		logger.Infof("Postponing scheduled instance snapshot for cluster %s because one is in progress", c.ClusterID)
		return nil
	}

	backupName, err := snapshot.CreateInstanceBackup(context.Background(), c, true)
	if err != nil {
		return errors.Wrap(err, "failed to create instance backup")
	}

	if err := store.GetStore().UpdateScheduledInstanceSnapshot(next.ID, backupName); err != nil {
		return errors.Wrap(err, "failed to update scheduled instance snapshot")
	}
	logger.Infof("Created instance backup %s from scheduled instance snapshot %s", backupName, next.ID)

	if len(pending) > 1 {
		err := store.GetStore().DeletePendingScheduledInstanceSnapshots(c.ClusterID)
		if err != nil {
			return errors.Wrap(err, "failed to delete pending scheduled instance snapshots")
		}
	}

	queued, err := nextScheduledInstanceSnapshot(c.ClusterID, c.SnapshotSchedule)
	if err != nil {
		return errors.Wrap(err, "failed to get next schedule")
	}

	if err := store.GetStore().CreateScheduledInstanceSnapshot(queued.ID, queued.ClusterID, queued.ScheduledTimestamp); err != nil {
		return errors.Wrap(err, "failed to create scheduled instance snapshot")
	}
	logger.Infof("Scheduled next instance snapshot %s", queued.ID)

	return nil
}

func nextScheduledApplicationSnapshot(appID string, cronExpression string) (*snapshottypes.ScheduledSnapshot, error) {
	cronSchedule, err := cron.ParseStandard(cronExpression)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse cron expression")
	}

	scheduledSnapshot := &snapshottypes.ScheduledSnapshot{
		AppID:              appID,
		ID:                 strings.ToLower(rand.String(32)),
		ScheduledTimestamp: cronSchedule.Next(time.Now()),
	}

	return scheduledSnapshot, nil
}

func nextScheduledInstanceSnapshot(clusterID string, cronExpression string) (*snapshottypes.ScheduledInstanceSnapshot, error) {
	cronSchedule, err := cron.ParseStandard(cronExpression)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse cron expression")
	}

	scheduledSnapshot := &snapshottypes.ScheduledInstanceSnapshot{
		ClusterID:          clusterID,
		ID:                 strings.ToLower(rand.String(32)),
		ScheduledTimestamp: cronSchedule.Next(time.Now()),
	}

	return scheduledSnapshot, nil
}
