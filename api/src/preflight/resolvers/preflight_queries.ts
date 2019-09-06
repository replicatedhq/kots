import _ from "lodash";
import { PreflightResult } from "../";
import { ReplicatedError } from "../../server/errors";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function PrefightQueries(stores: Stores) {
  return {
    async listPreflightResults(root: any, args: any, context: Context): Promise<PreflightResult[]> {
      const { watchId, slug } = args;

      let preflights;

      if (watchId) {
        preflights = await stores.preflightStore.getPreflightsResultsByWatchId(watchId);
      }

      if (slug) {
        preflights = await stores.preflightStore.getPreflightsResultsBySlug(slug);
      }

      if (!preflights) {
        throw new ReplicatedError("listPreflightResults requires either 'watchId' or 'slug");
      }

      return preflights;
    },

    async getKotsPreflightResult(root: any, args: any, context: Context): Promise<PreflightResult> {
      const { appSlug, clusterSlug, sequence } = args;

      const appId = await stores.kotsAppStore.getIdFromSlug(appSlug);
      const clusterId = await stores.clusterStore.getIdFromSlug(clusterSlug);

      const result = await stores.preflightStore.getKotsPreflightResult(appId, clusterId, sequence);
      return result;
    },

    async getLatestKotsPreflightResult(root: any, args: any, context: Context): Promise<PreflightResult> {
      const result = await stores.preflightStore.getLatestKotsPreflightResult();
      return result;
    }
  };
}
