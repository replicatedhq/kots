import * as _ from "lodash";
import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";
import { Context } from "../../context";
import {
  RestoreDetail,
} from "../snapshot";
import { Phase } from "../velero";
import { SnapshotConfig } from "../snapshot_config";
import { VeleroClient } from "./veleroClient";
import { parseTTL } from "../backup";

export function SnapshotQueries(stores: Stores, params: Params) {
  // tslint:disable-next-line max-func-body-length
  return {
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
  };
}
