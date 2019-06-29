import { HelmChart } from "./";
import { ReplicatedError } from "../server/errors";
import * as pg from "pg";

export class HelmChartStore {
  constructor(
    private readonly pool: pg.Pool,
  ) {
  }

  public async listHelmChartsInCluster(clusterId: string): Promise<HelmChart[]> {
    const q = `select id from helm_chart where cluster_id = $1`;
    const v = [
      clusterId,
    ];
    const result = await this.pool.query(q, v);
    const helmCharts: HelmChart[] = [];
    for (const row of result.rows) {
      const helmChart = await this.getChart(row.id);
      helmCharts.push(helmChart);
    }
    return helmCharts;
  }

  public async getChart(chartId: string): Promise<HelmChart> {
    const q = `select id, cluster_id, helm_name, namespace, version, first_deployed_at, last_deployed_at, is_deleted, chart_version, app_version
      from helm_chart where id = $1`;
    const v = [
      chartId,
    ];

    const result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`No helm chart found for ID ${chartId}. Check your watch dashboard to see if you have already unforked this chart`);
    }
    const row = result.rows[0];
    const helmChart = new HelmChart();
    helmChart.id = row.id;
    helmChart.clusterId = row.cluster_id;
    helmChart.helmName = row.helm_name;
    helmChart.namespace = row.namespace;
    helmChart.version = row.version;
    helmChart.firstDeployedAt = row.first_deployed_at;
    helmChart.lastDeployedAt = row.last_deployed_at;
    helmChart.isDeleted = row.is_deleted;
    helmChart.chartVersion = row.chart_version;
    helmChart.appVersion = row.app_version;

    return helmChart;
  }

}
