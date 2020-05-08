import randomstring from "randomstring";
import Cron from "cron-converter";
import { Pool } from "pg";
import { logger } from "../server/logger";
import { getPostgresPool } from "../util/persistence/db";
import { backup } from "./backup";
import { KotsApp } from "../kots_app/kots_app";
import { Stores } from "../schema/stores";
import { ScheduledSnapshot } from "./snapshot";
import { VeleroClient } from "./resolvers/veleroClient";

export class SnapshotScheduler {
  constructor(
    private stores: Stores,
    private pool: Pool,
  ) {}
  
  run() {
    setInterval(this.scheduleLoop.bind(this), 60000);
  }

  async scheduleLoop() {
    try {
      const apps = await this.stores.kotsAppStore.listApps();
      for (const app of apps) {
        if (app.restoreInProgressName) {
          continue;
        }
        await this.handleApp(app);
      }
    } catch (e) {
      logger.error(e);
    }
  }

  // tslint:disable-next-line cyclomatic-complexity
  async handleApp(app: KotsApp) {
    if (!app.snapshotSchedule) {
      return;
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
    const pending = await this.stores.snapshotsStore.listPendingScheduledSnapshots(app.id);
    const [next] = pending;
    if (!next) {
      logger.warn(`No pending snapshots scheduled for app ${app.id} with schedule ${app.snapshotSchedule}. Queueing one.`);
      const queued = nextScheduled(app.id, app.snapshotSchedule);
      await this.stores.snapshotsStore.createScheduledSnapshot(queued);
      return;
    }
    if (new Date(next.scheduledTimestamp).valueOf() > Date.now()) {
      logger.debug(`Not yet time to snapshot app ${app.id}`);
      return;
    }

    const velero = new VeleroClient("velero"); // TODO namespace
    const hasUnfinished = await velero.hasUnfinishedBackup(app.id);
    if (hasUnfinished) {
      logger.info(`Postponing scheduled snapshot for ${app.id} because one is in progress`);
      return;
    }

    const client = await this.pool.connect();
    try {
      await client.query("BEGIN");

      logger.info(`Acquiring lock on scheduled snapshot ${next.id}`);
      const acquiredLock = await this.stores.snapshotsStore.lockScheduledSnapshot(client, next.id);
      if (!acquiredLock) {
        logger.info(`Failed to lock scheduled snapshot ${next.id}`);
        client.query("ROLLBACK");
        return;
      }

      const b = await backup(this.stores, app.id, true);
      await this.stores.snapshotsStore.updateScheduledSnapshot(client, next.id, b.metadata.name!);
      logger.info(`Created backup ${b.metadata.name} from scheduled snapshot ${next.id}`);

      if (pending.length > 1) {
        await this.stores.snapshotsStore.deletePendingScheduledSnapshots(app.id, client);
      }
      const queued = nextScheduled(app.id, app.snapshotSchedule);
      await this.stores.snapshotsStore.createScheduledSnapshot(queued, client);
      logger.info(`Scheduled next snapshot ${queued.id}`);

      await client.query("COMMIT");
    } catch (e) {
      await client.query("ROLLBACK");
      throw e;
    } finally {
      client.release();
    }
  }
};

export function nextScheduled(appId: string, cronExpression: string): ScheduledSnapshot {
  const cron = new Cron();

  return {
    appId,
    id: randomstring.generate({ capitalization: "lowercase" }),
    scheduledTimestamp: cron.fromString(cronExpression).schedule().next(),
  };
}
