import { CreateUpdateSessionMutationArgs, UpdateSession } from "../../generated/types";
import { Context } from "../../context";

export function UpdateMutations(stores: any) {
  return {
    async createUpdateSession(root: any, { watchId }: CreateUpdateSessionMutationArgs, context: Context): Promise<UpdateSession> {
      const updateSession = await stores.updateStore.createUpdateSession(context.session.userId, watchId);
      const deployedUpdateSession = await stores.updateStore.deployUpdateSession(updateSession.id!);
      return deployedUpdateSession;
    }
  }
}
