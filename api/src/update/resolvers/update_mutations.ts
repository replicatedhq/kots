import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function UpdateMutations(stores: Stores) {
  return {
    async createUpdateSession(root: any, args: any, context: Context) {
      const watch = await context.getWatch(args.watchId);
      const updateSession = await stores.updateStore.createUpdateSession(context.session.userId, watch.id);
      const deployedUpdateSession = await stores.updateStore.deployUpdateSession(updateSession.id);
      return deployedUpdateSession;
    }
  }
}
