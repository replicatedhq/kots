import { HelmChart } from "./";
import * as pg from "pg";

export class HelmChartStore {
  constructor(
    private readonly pool: pg.Pool,
  ) {
  }

  public async listHelmChartsInCluster(clusterId: string): Promise<HelmChart[]> {
    const q = `select id, helm_name, namespace, version, first_deployed_at, last_deployed_at, is_deleted, app_version, chart_version from helm_chart where cluster_id = $1`;
    const v = [
      clusterId,
    ];
    const result = await this.pool.query(q, v);
    const helmCharts: HelmChart[] = [];
    for (const row of result.rows) {
      const helmChart: HelmChart = {
        id: row.id,
        clusterID: clusterId,
        helmName: row.helm_name,
        namespace: row.namespace,
        version: row.version,
        firstDeployedAt: new Date(row.first_deployed_at),
        lastDeployedAt: new Date(row.last_deployed_at),
        isDeleted: row.is_deleted,
        chartVersion: row.chart_version,
        appVersion: row.app_version,
      };

      helmCharts.push(helmChart);
    }

    return helmCharts;
  }

}
