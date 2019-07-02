import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function PendingQueries(stores: Stores) {
  return {
    async listPendingInitSessions(root: any, args: any, context: Context) {
      const pendingInitSessions = await stores.pendingStore.listPendingInitSessions(context.session.userId);
      return pendingInitSessions.map((pendingInitSession) => {
        return {
          id: pendingInitSession.id,
          upstreamURI: pendingInitSession.upstreamURI,
          title: pendingInitSession.title,
        };
      });
    },

    async getPendingIniSession(root: any, args: any, context: Context) {
      const session = await stores.pendingStore.getPendingInitSession(args.id, context.session.userId);
      return {
        id: session.id,
        upstreamURI: session.upstreamURI,
        title: session.title
      }
    },

    async searchPendingInitSessions(root: any, args: any, context: Context) {
      const { title } = args;
      const pendingInitSessions = await stores.pendingStore.searchPendingInitSessions(context.session.userId, title);
      return pendingInitSessions.map((pendingInitSession) => {
        return {
          id: pendingInitSession.id,
          upstreamURI: pendingInitSession.upstreamURI,
          title: pendingInitSession.title,
        };
      });
    },
  }
}
