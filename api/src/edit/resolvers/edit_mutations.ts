import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function EditMutations(stores: Stores) {
  return {
    async createEditSession(root: any, { watchId }: any, context: Context) {
      const editSession = await stores.editStore.createEditSession(context.session.userId, watchId);
      const deployedEditSession = await stores.editStore.deployEditSession(editSession.id);
      return deployedEditSession;
    }
  }
}
