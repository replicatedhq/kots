import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";
import { Context } from "../../context";

export function KurlMutations(stores: Stores, params: Params) {
  return {
    async drainNode(root: any, { name }, context: Context) {
      context.requireSingleTenantSession();

      return false;
    },

    async deleteNode(root: any, { name }, context: Context) {
      context.requireSingleTenantSession();

      return false;
    }
  }
}
