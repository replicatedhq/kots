import * as cronstrue from "cronstrue";
import * as _ from "lodash";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { VeleroClient } from "./veleroClient";
import { ReplicatedError } from "../../server/errors";
import { formatTTL } from "../backup";
import { nextScheduled } from "../schedule";

export function SnapshotMutations(stores: Stores) {
  // tslint:disable-next-line max-func-body-length cyclomatic-complexity
  return {
    async saveSnapshotConfig(root: any, args: any, context: Context): Promise<void> {
      context.requireSingleTenantSession();

      const {
        appId,
        inputValue: retentionQuantity,
        inputTimeUnit: retentionUnit,
        userSelected: scheduleSelected,
        schedule: scheduleExpression,
        autoEnabled,
      } = args;

      const app = await stores.kotsAppStore.getApp(appId);

      const retention = formatTTL(retentionQuantity, retentionUnit);
      if (app.snapshotTTL !== retention) {
        await stores.kotsAppStore.updateAppSnapshotTTL(appId, retention);
      }

      if (!autoEnabled) {
        await stores.kotsAppStore.updateAppSnapshotSchedule(app.id, null);
        await stores.snapshotsStore.deletePendingScheduledSnapshots(app.id);
        return;
      }

      try {
        cronstrue.toString(scheduleExpression);
      } catch(e) {
        throw new ReplicatedError(`Invalid snapshot schedule: ${scheduleExpression}`);
      }
      if (scheduleExpression.split(" ").length > 5) {
        throw new ReplicatedError("Snapshot schedule expression does not support seconds or years");
      }

      if (scheduleExpression !== app.snapshotSchedule) {
        await stores.snapshotsStore.deletePendingScheduledSnapshots(app.id);
        await stores.kotsAppStore.updateAppSnapshotSchedule(app.id, scheduleExpression);
        const queued = nextScheduled(app.id, scheduleExpression);
        await stores.snapshotsStore.createScheduledSnapshot(queued);
      }
    },

    async cancelRestore(root: any, args: any, context: Context): Promise<void> {
      await stores.kotsAppStore.updateAppRestoreReset(args.appId);
    },

    async deleteSnapshot(root: any, args: any, context: Context): Promise<void> {
      context.requireSingleTenantSession();
      const velero = new VeleroClient("velero"); // TODO namespace
      await velero.deleteSnapshot(args.snapshotName);
    },
  };
}
