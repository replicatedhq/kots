import * as _ from "lodash";
import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";
import { Context } from "../../context";
import {
  kotsClusterIdKey,
  kotsAppSequenceKey,
  RestoreDetail,
  Snapshot,
  SnapshotDetail,
  SnapshotTrigger,
  SnapshotHookPhase,
} from "../snapshot";
import { Phase } from "../velero";
import { SnapshotConfig, AzureCloudName, SnapshotProvider } from "../snapshot_config";
import { VeleroClient } from "./veleroClient";
import { readSchedule } from "../schedule";
import { parseTTL, formatTTL } from "../backup";
import { ReplicatedError } from "../../server/errors";

export function SnapshotQueries(stores: Stores, params: Params) {
  // tslint:disable-next-line max-func-body-length
  return {
    async snapshotConfig(root: any, args: any, context: Context): Promise<SnapshotConfig> {
      context.requireSingleTenantSession();

      const velero = new VeleroClient("velero"); // TODO namespace
      const store = await velero.readSnapshotStore();
      const schedule = await readSchedule(args.slug);
      const appId = await stores.kotsAppStore.getIdFromSlug(args.slug);
      const app = await stores.kotsAppStore.getApp(appId);

      let ttl = {
        inputValue: "1",
        inputTimeUnit: "month",
        converted: "720h",
      };
      if (app.snapshotTTL) {
        const { quantity, unit } = parseTTL(app.snapshotTTL);
        ttl = {
          inputValue: quantity.toString(),
          inputTimeUnit: unit,
          converted: app.snapshotTTL,
        };
      }

      return {
        autoEnabled: !!schedule,
        autoSchedule: schedule ? { userSelected: schedule.selection, schedule: schedule.schedule } : { userSelected: "weekly", schedule: "0 0 * * MON" },
        ttl,
        store,
      };
    },

    async listSnapshots(root: any, args: any, context: Context): Promise<Array<Snapshot>> {
      context.requireSingleTenantSession();

      const { slug } = args;
      const client = new VeleroClient("velero"); // TODO namespace
      const snapshots = await client.listSnapshots();

      // TODO filter earlier
      return _.filter(snapshots, { appSlug: slug });
    },

    async snapshotDetail(root: any, args: any, context: Context): Promise<SnapshotDetail> {
      context.requireSingleTenantSession();
      const { slug, id } = args;
      const client = new VeleroClient("velero"); // TODO namespace
      return await client.getSnapshotDetail(id);
    },

    // tslint:disable-next-line cyclomatic-complexity
    async restoreDetail(root: any, args: any, context: Context): Promise<RestoreDetail> {
      context.requireSingleTenantSession();

      const { appId } = args;
      const { restoreInProgressName: name } = await stores.kotsAppStore.getApp(appId);
      if (!name) {
        throw new ReplicatedError("No restore is in progress");
      }
      const velero = new VeleroClient("velero"); // TODO namespace
      const restore = await velero.readRestore(name);
      if (!restore) {
        return {
          name,
          phase: Phase.New,
          volumes: [],
          errors: [],
          warnings: [],
        };
      }

      const volumes = await velero.listRestoreVolumes(name);
      const detail: RestoreDetail = {
        name,
        phase: restore.status ? restore.status.phase : Phase.New,
        volumes,
        errors: [],
        warnings: [],
      };

      if (detail.phase === Phase.Completed || detail.phase === Phase.PartiallyFailed || detail.phase === Phase.Failed) {
        const results = await velero.getRestoreResults(name);

        _.each(results.warnings.namespaces, (warnings, namespace) => {
          _.each(warnings, (warning) => {
            detail.warnings.push({
              message: warning,
              namespace,
            });
          });
        });

        _.each(results.errors.namespaces, (errors, namespace) => {
          _.each(errors, (error) => {
            detail.errors.push({
              message: error,
              namespace,
            });
          });
        });
      }

      return detail;
    },
  };
}
