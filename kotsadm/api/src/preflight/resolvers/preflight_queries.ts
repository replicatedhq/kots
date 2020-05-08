import _ from "lodash";
import { PreflightResult } from "../";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function PrefightQueries(stores: Stores) {
  return {
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
    },

    async getPreflightCommand(root: any, args: any, context: Context): Promise<string> {
      const { appSlug, clusterSlug, sequence } = args;
      return await stores.preflightStore.getPreflightCommand(appSlug, clusterSlug, sequence);
    }
  };
}
