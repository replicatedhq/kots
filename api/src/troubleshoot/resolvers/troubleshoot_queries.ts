import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import * as _ from "lodash";
import { ReplicatedError } from "../../server/errors";

export function TroubleshootQueries(stores: Stores) {
  return {
    async watchCollectors(root: any, { watchId }, context: Context) {
      const collector = await stores.troubleshootStore.getPreferedWatchCollector(watchId);

      return {
        spec: collector.spec,
        hydrated: collector.spec,
      };
    },

    async listSupportBundles(root: any, { watchSlug }, context: Context) {
      const watch = await context.findWatch(watchSlug);
      const supportBundles = await stores.troubleshootStore.listSupportBundles(watch.id);
      return _.map(supportBundles, (supportBundle) => {
        return supportBundle.toSchema();
      });
    },

    async getSupportBundle(root: any, { id }, context: Context) {
      const supportBundle = await stores.troubleshootStore.getSupportBundle(id);
      const watch = context.getWatch(supportBundle.watchId);
      if (!watch) {
        throw new ReplicatedError("not found");
      }

      return supportBundle.toSchema();
    },
  };
}
