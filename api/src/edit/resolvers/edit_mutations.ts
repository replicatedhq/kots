import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function EditMutations(stores: Stores) {
  return {
    async createEditSession(root: any, args: any, context: Context) {
      const watch = await context.getWatch(args.watchId);
      const editSession = await stores.editStore.createEditSession(context.session.userId, watch.id);
      const deployedEditSession = await stores.editStore.deployEditSession(editSession.id);
      return deployedEditSession;
    }
  }
}
