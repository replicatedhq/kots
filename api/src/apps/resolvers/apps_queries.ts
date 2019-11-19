import _ from "lodash";
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
  }
}
