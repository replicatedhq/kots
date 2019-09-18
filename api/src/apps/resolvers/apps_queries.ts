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
      if (params.enableShip) {
        result.kotsApps = [];
        result.watches = async () => (await stores.watchStore.listWatches(context.session.userId)).map(watch => watch.toSchema(root, stores, context));
        result.pendingUnforks = async () => {
          const clusters = await stores.clusterStore.listClusters(context.session.userId);
          let helmCharts: HelmChart[] = [];
          for (const cluster of clusters) {
            const clusterCharts = await stores.helmChartStore.listHelmChartsInCluster(cluster.id);
            helmCharts = helmCharts.concat(clusterCharts);
          }
          return helmCharts.map(chart => chart.toSchema());
        };
      }
      if (params.enableKots) {
        result.kotsApps = async () => (await stores.kotsAppStore.listInstalledKotsApps(context.session.userId)).map(async (kotsApp) => {
          const downstreams = await stores.clusterStore.listClustersForKotsApp(kotsApp.id);
          return kotsApp.toSchema(downstreams, stores);
        });
        result.watches = [];
        result.pendingUnforks = [];
      }

      return result;
    },

    // async searchApps(root: any, args: any, context: Context): Promise<Watch[]> {
    //   const watches = await stores.watchStore.searchWatches(context.session.userId, args.watchName);
    //   return watches.map(watch => watch.toSchema(root, stores, context));
    // },

  }
}
