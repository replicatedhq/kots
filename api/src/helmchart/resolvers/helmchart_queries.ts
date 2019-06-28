import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { HelmChart } from "../";

export function HelmChartQueries(stores: Stores) {
  return {
    async listHelmCharts(root: any, args: any, context: Context) {
      const clusters = await stores.clusterStore.listClusters(context.session.userId);

      let helmCharts: HelmChart[] = [];
      for (const cluster of clusters) {
        const clusterCharts = await stores.helmChartStore.listHelmChartsInCluster(cluster.id);

        helmCharts = helmCharts.concat(clusterCharts);
      }

      return helmCharts.map((helmChart) => {
        return {
          ...helmChart,
        };
      });
    },

    async getHelmChart(root: any, args: any, context: Context) {
      const chart = await stores.helmChartStore.getChartInCluster(args.id, args.clusterId);
      return chart;
    }
  }
}
