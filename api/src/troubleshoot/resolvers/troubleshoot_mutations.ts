import _ from "lodash";

import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";
import { Context } from "../../context";

export function TroubleshootMutations(stores: Stores, params: Params) {
  return {
    async collectSupportBundle(root, { appId, clusterId }, context: Context) {
      const app = await context.getApp(appId);

      await stores.troubleshootStore.queueSupportBundleCollection(appId, clusterId);
    }
  }
}
