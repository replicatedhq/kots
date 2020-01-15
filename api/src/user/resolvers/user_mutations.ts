import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";

export function UserMutations(stores: Stores, params: Params) {
  return {
    async logout(root: any, args: any, context: Context): Promise<void> {
      await stores.sessionStore.deleteSession(context.session.sessionId);
    },
  }
}
