import _ from "lodash";
import { PreflightResult } from "../";
// import { ReplicatedError } from "../../server/errors";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function PrefightQueries(stores: Stores) {
  return {
    async listPreflightResults(root: any, args: any, context: Context): Promise<PreflightResult[]> {
      const { watchId } = args;
      const preflights = await stores.preflightStore.getPreflightsResultsByWatchId(watchId);

      return preflights;
    },
  };
}
