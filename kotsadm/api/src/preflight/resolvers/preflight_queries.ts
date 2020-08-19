import _ from "lodash";
import { PreflightResult } from "../";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function PrefightQueries(stores: Stores) {
  return {
    async getPreflightCommand(root: any, args: any, context: Context): Promise<string> {
      const { appSlug, clusterSlug, sequence } = args;
      return await stores.preflightStore.getPreflightCommand(appSlug, clusterSlug, sequence);
    }
  };
}
