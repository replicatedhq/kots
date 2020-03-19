import * as _ from "lodash";
import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";
import { Context } from "../../context";
import {
  RestoreDetail,
  Snapshot,
  SnapshotDetail
} from "../snapshot";
import { Phase } from "../velero";
import { SnapshotConfig, SnapshotSettings } from "../snapshot_config";
import { VeleroClient } from "./veleroClient";
import { parseTTL } from "../backup";
import { logger } from "../../server/logger";

export function SnapshotQueries(stores: Stores, params: Params) {
  // tslint:disable-next-line max-func-body-length
  return {
    async isVeleroInstalled(root: any, args: any, context: Context): Promise<boolean> {
      context.requireSingleTenantSession();
      
      const velero = new VeleroClient("velero");
      return velero.isVeleroInstalled();
    },

    async snapshotConfig(root: any, args: any, context: Context): Promise<SnapshotConfig> {
      context.requireSingleTenantSession();

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
        }
      }

      return {
        autoEnabled: !!app.snapshotSchedule,
        autoSchedule: app.snapshotSchedule ? { schedule: app.snapshotSchedule } : { schedule: "0 0 * * MON" },
        ttl
      };
    },

    async snapshotSettings(root: any, args: any, context: Context): Promise<SnapshotSettings> {
      context.requireSingleTenantSession();

      const velero = new VeleroClient("velero"); // TODO namespace
      const store = await velero.readSnapshotStore();

      return {
        store,
      };
    },

    async listSnapshots(root: any, args: any, context: Context): Promise<Snapshot[]> {
      context.requireSingleTenantSession();

      const { slug } = args;
      const velero = new VeleroClient("velero"); // TODO namespace
      await velero.maybeCreateAppBackend(slug);

      return velero.listSnapshots(slug);
    },

    async snapshotDetail(root: any, args: any, context: Context): Promise<SnapshotDetail> {
      context.requireSingleTenantSession();
      const { slug, id } = args;
      const client = new VeleroClient("velero"); // TODO namespace
      const detail = await client.getSnapshotDetail(id);
      return detail;
    },

    // tslint:disable-next-line cyclomatic-complexity
    async restoreDetail(root: any, args: any, context: Context): Promise<RestoreDetail> {
      context.requireSingleTenantSession();

      const { appId, restoreName: name } = args;
      const { restoreInProgressName } = await stores.kotsAppStore.getApp(appId);
      const active = !!restoreInProgressName && restoreInProgressName === name;
      const velero = new VeleroClient("velero"); // TODO namespace
      const restore = await velero.readRestore(name);
      if (!restore) {
        return {
          name,
          active,
          phase: Phase.New,
          volumes: [],
          errors: [],
          warnings: [],
        };
      }

      const volumes = await velero.listRestoreVolumes(name);
      const detail: RestoreDetail = {
        name,
        active,
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
