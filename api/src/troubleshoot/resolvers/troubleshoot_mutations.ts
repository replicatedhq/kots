import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { SupportBundle } from "../";

export function TroubleshootMutations(stores: Stores) {
  return {
    async uploadSupportBundle(root: any, args: any, context: Context): Promise<SupportBundle> {
      // TODO

      return {};
    },

    async markSupportBundleUploaded(root: any, args: any, context: Context): Promise<SupportBundle> {
      // TODO

      return {};
    },
  }
}
