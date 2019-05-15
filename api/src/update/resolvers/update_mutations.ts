import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function UpdateMutations(stores: Stores) {
  return {
    async createUpdateSession(root: any, { watchId }: any, context: Context) {
      const updateSession = await stores.updateStore.createUpdateSession(context.session.userId, watchId);
      const deployedUpdateSession = await stores.updateStore.deployUpdateSession(updateSession.id);
      return deployedUpdateSession;
    }
  }
}
