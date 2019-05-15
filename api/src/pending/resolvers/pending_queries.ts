import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function PendingQueries(stores: Stores) {
  return {
    async listPendingInit(root: any, args: any, context: Context) {
      const pendingInitSessions = await stores.pendingStore.listPendingInitSessions(context.session.userId);
      return pendingInitSessions.map((pendingInitSession) => {
        return {
          id: pendingInitSession.id,
          upstreamURI: pendingInitSession.upstreamURI,
          title: pendingInitSession.title,
        };
      });
    }
  }
}
