import { HelmChart } from "./";
import { ReplicatedError } from "../server/errors";
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

  public async getChartInCluster(chartId: string, clusterId: string): Promise<HelmChart> {
    const q = `select id, helm_name, namespace, version, first_deployed_at, last_deployed_at, is_deleted, app_version, chart_version from helm_chart where id = $1 and cluster_id = $2`;
    const v = [
      chartId,
      clusterId,
    ];
    const result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`No helm chart found for ID ${chartId}. Check your watch dashboard to see if you have already unforked this chart`);
    }
    
    return result.rows[0]; 
  }

}
