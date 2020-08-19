import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import _ from "lodash";
import { ReplicatedError } from "../../server/errors";

export function TroubleshootQueries(stores: Stores) {
  return {
    async listSupportBundles(root: any, { watchSlug }, context: Context) {
      const appId = await stores.kotsAppStore.getIdFromSlug(watchSlug);
      const app = await context.getApp(appId);
      let supportBundles = await stores.troubleshootStore.listSupportBundles(app.id);

      return _.map(supportBundles, async (supportBundle) => {
        return supportBundle.toSchema();
      });
    },

    async getSupportBundle(root: any, { watchSlug }, context: Context) {
      const supportBundle = await stores.troubleshootStore.getSupportBundle(watchSlug);
      // TODO: Write some method that triest to fetch an app.
      // let app = await stores.kotsAppStore.getIdFromSlug(watchSlug);
      if (!supportBundle) {
        throw new ReplicatedError("not found");
      }

      return supportBundle.toSchema();
    },
  };
}
