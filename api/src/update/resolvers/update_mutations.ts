import { Context } from "../../context";

export function UpdateMutations(stores: any) {
  return {
    async createUpdateSession(root: any, { watchId }: any, context: Context) {
      const updateSession = await stores.updateStore.createUpdateSession(context.session.userId, watchId);
      const deployedUpdateSession = await stores.updateStore.deployUpdateSession(updateSession.id!, context.session.userId);
      return deployedUpdateSession;
    }
  }
}
