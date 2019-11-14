import _ from "lodash";
import { HelmChart } from "../../helmchart";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";

export function AppsQueries(stores: Stores) {
  return {
    async listApps(root: any, args: any, context: Context) {
      const params = await Params.getParams();
      const result: any = {};

      result.kotsApps = async () => (await stores.kotsAppStore.listInstalledKotsApps(context.session.userId)).map(async (kotsApp) => {
        const downstreams = await stores.clusterStore.listClustersForKotsApp(kotsApp.id);
        return kotsApp.toSchema(downstreams, stores);
      });
      result.pendingUnforks = [];

      return result;
    },

    // async searchApps(root: any, args: any, context: Context): Promise<Watch[]> {
    //   const watches = await stores.watchStore.searchWatches(context.session.userId, args.watchName);
    //   return watches.map(watch => watch.toSchema(root, stores, context));
    // },

  }
}
