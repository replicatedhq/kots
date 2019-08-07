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
  };
}
