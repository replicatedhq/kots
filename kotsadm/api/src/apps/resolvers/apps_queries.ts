import _ from "lodash";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function AppsQueries(stores: Stores) {
  return {
    async listApps(root: any, args: any, context: Context) {
      const result: any = {};

      result.kotsApps = async () => (await stores.kotsAppStore.listInstalledKotsApps(context.session.userId)).map(async (kotsApp) => {
        const downstreams = await stores.clusterStore.listClustersForKotsApp(kotsApp.id);
        return kotsApp.toSchema(downstreams, stores);
      });
      result.pendingUnforks = [];

      return result;
    },

    async getGitOpsRepo(root: any, args: any, context: Context) {
      return await stores.kotsAppStore.getGitOpsRepo();
    },
  }
}
