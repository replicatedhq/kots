import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { SupportBundle } from "../";

export function TroubleshootQueries(stores: Stores) {
  return {
    async listSupportBundles(root: any, { watchId }, context: Context): Promise<SupportBundle[]> {
      return await stores.troubleshootStore.listSupportBundles(watchId);
    },

    async getSupportBundle(root: any, { id }, context: Context): Promise<SupportBundle> {
      return await stores.troubleshootStore.getSupportBundle(id);
    },
  };
}
