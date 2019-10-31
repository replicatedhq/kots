import { Stores } from "../../schema/stores";
import { Context } from "../../context";
import { ReplicatedError } from "../../server/errors";
import { State, KotsAppStatusSchema } from "../kots_app_status";
import { MetricChart } from "../../monitoring";
import { MetricGraph, AxisFormat, ApplicationSpec } from "../kots_app_spec";
import { logger } from "../../server/logger";

interface KotsAppDashboard {
  appStatus: () => Promise<KotsAppStatusSchema>;
  metrics: () => Promise<MetricChart[]>;
  prometheusAddress: () => Promise<string>;
}

export function KotsDashboardQueries(stores: Stores) {
  return {
    async getKotsAppDashboard(root: any, args: any, context: Context): Promise<KotsAppDashboard> {
      return {
        appStatus: () => getKotsAppStatus(stores, root, args, context),
        metrics: async () => {
          try {
            return await getKotsAppMetricCharts(stores, root, args, context);
          } catch(err) {
            logger.error("[getKotsAppDashboard] - Unable to retrieve metrics charts", err);
            return [];
          }
        },
        prometheusAddress: async () => (await stores.paramsStore.getParam("PROMETHEUS_ADDRESS")) || this.params.prometheusAddress,
      }
    },
  };
}

async function getKotsAppStatus(stores: Stores, root: any, args: any, context: Context): Promise<KotsAppStatusSchema> {
  const { slug } = args;
  const appId = await stores.kotsAppStore.getIdFromSlug(slug)
  const app = await context.getApp(appId);
  try {
    const appStatus = await stores.kotsAppStatusStore.getKotsAppStatus(app.id);
    return appStatus.toSchema();
  } catch (err) {
    if (ReplicatedError.isNotFound(err)) {
      return {
        appId,
        updatedAt: new Date(),
        resourceStates: [],
        state: State.Missing,
      };
    }
    throw err;
  }
}

const DefaultMetricGraphs: MetricGraph[] = [
  {
    title: "Disk Usage",
    queries: [{
      query: `sum((node_filesystem_size_bytes{job="node-exporter",fstype!="",instance!=""} - node_filesystem_avail_bytes{job="node-exporter", fstype!=""})) by (instance)`,
      legend: "Used: {{ instance }}",
    },
    {
      query: `sum((node_filesystem_avail_bytes{job="node-exporter",fstype!="",instance!=""})) by (instance)`,
      legend: "Available: {{ instance }}",
    }],
    yAxisFormat: AxisFormat.Bytes,
    yAxisTemplate: "{{ value }} bytes",
  },
  {
    title: "CPU Usage",
    query: `sum(rate(container_cpu_usage_seconds_total{namespace="default",container_name!="POD",pod_name!=""}[5m])) by (pod_name)`,
    legend: "{{ pod_name }}",
    yAxisFormat: AxisFormat.Short,
  },
  {
    title: "Memory Usage",
    query: `sum(container_memory_usage_bytes{namespace="default",container_name!="POD",pod_name!=""}) by (pod_name)`,
    legend: "{{ pod_name }}",
    yAxisFormat: AxisFormat.Short,
  },
];

async function getKotsAppMetricCharts(stores: Stores, root: any, args: any, context: Context): Promise<MetricChart[]> {
  const { slug, clusterId } = args;
  const appId = await stores.kotsAppStore.getIdFromSlug(slug)
  const app = await context.getApp(appId);
  let kotsAppSpec: ApplicationSpec | undefined;
  try {
    kotsAppSpec = await app.getKotsAppSpec(clusterId, stores.kotsAppStore)
  } catch (err) {
    logger.error("[getKotsAppMetricCharts] - Unable to retrieve kots app spec", err);
  }
  const graphs = kotsAppSpec && kotsAppSpec.graphs ? kotsAppSpec.graphs : DefaultMetricGraphs;
  return await stores.metricStore.getKotsAppMetricCharts(graphs);
}
