package snapshotscheduler

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/snapshot"
	snapshottypes "github.com/replicatedhq/kots/kotsadm/pkg/snapshot/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"k8s.io/apimachinery/pkg/util/rand"

	cron "github.com/robfig/cron/v3"
)

func Start() error {
	logger.Debug("starting snapshot scheduler")

	startLoop(scheduleLoop, 60)

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

func scheduleLoop() {
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
	* If there are multiple replicas running of this api, or if this loop takes longer than 1
	* minute, then there will be concurrent reads/writes on this table. Listing pending snapshots
	* does not lock any rows and may return a row that is locked by another transaction. Before
	* taking a lock on the row, first check that it's not scheduled for a time in the future, then
	* check that there is not already another snapshot in progress for the app. If both of those
	* checks pass than attempt to acquire a lock on the row. Acquiring a lock uses `SKIP LOCKED`
	* so it does not wait if another transaction has already acquired a lock on the row.
	*
	* If the lock is acuired, create the Backup CR for velero, save the Backup name to the row to
	* mark that it has been handled, then schedule the next snapshot from the app's cron schedule
	* expression in the same transaction.
	 */

	pending, err := store.GetStore().ListPendingScheduledSnapshots(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to list pending scheduled snapshots")
	}

	if len(pending) == 0 {
		logger.Infof("No pending snapshots scheduled for app %s with schedule %s. Queueing one.", a.ID, a.SnapshotSchedule)
		queued, err := nextScheduled(a.ID, a.SnapshotSchedule)
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

	hasUnfinished, err := snapshot.HasUnfinishedBackup(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to to check if app has unfinished backups")
	}
	if hasUnfinished {
		logger.Infof("Postponing scheduled snapshot for %s because one is in progress", a.ID)
		return nil
	}

	backup, err := snapshot.CreateBackup(a, true)
	if err != nil {
		return errors.Wrap(err, "failed to create backup")
	}

	if err := store.GetStore().UpdateScheduledSnapshot(next.ID, backup.ObjectMeta.Name); err != nil {
		return errors.Wrap(err, "failed to update scheduled snapshot")
	}
	logger.Infof("Created backup %s from scheduled snapshot %s", backup.ObjectMeta.Name, next.ID)

	if len(pending) > 1 {
		err := store.GetStore().DeletePendingScheduledSnapshots(a.ID)
		if err != nil {
			return errors.Wrap(err, "failed to delete pending scheduled snapshots")
		}
	}

	queued, err := nextScheduled(a.ID, a.SnapshotSchedule)
	if err != nil {
		return errors.Wrap(err, "failed to get next schedule")
	}

	if err := store.GetStore().CreateScheduledSnapshot(queued.ID, queued.AppID, queued.ScheduledTimestamp); err != nil {
		return errors.Wrap(err, "failed to create scheduled snapshot")
	}
	logger.Infof("Scheduled next snapshot %s", queued.ID)

	return nil
}

func nextScheduled(appID string, cronExpression string) (*snapshottypes.ScheduledSnapshot, error) {
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
