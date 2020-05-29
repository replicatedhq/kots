import pg from "pg";

import { Params } from "../server/params";
import { PreflightResult } from "./";
import { ReplicatedError } from "../server/errors";
import { logger } from "../server/logger";

interface PreflightParams {
  url: string
  ignorePermissions: boolean
}

interface PreflightParams {
  url: string
  ignorePermissions: boolean
}

export class PreflightStore {
  constructor(
    private readonly pool: pg.Pool
  ) {}

  async getKotsPreflightResult(appId: string, clusterId: string, sequence: number): Promise<PreflightResult> {
    const q =
      `SELECT
        app_downstream_version.preflight_result,
        app_downstream_version.preflight_result_created_at,
        app.slug as app_slug,
        cluster.slug as cluster_slug
      FROM app_downstream_version
        INNER JOIN app ON app_downstream_version.app_id = app.id
        INNER JOIN cluster ON app_downstream_version.cluster_id = cluster.id
      WHERE
        app_downstream_version.app_id = $1 AND
        app_downstream_version.cluster_id = $2 AND
        app_downstream_version.sequence = $3`;

    const v = [
      appId,
      clusterId,
      sequence,
    ];

    const result = await this.pool.query(q,v);

    if (result.rows.length === 0) {
      throw new Error(`Couldn't find preflight spec with appId: ${appId}, clusterId: ${clusterId}, sequence: ${sequence}`);
    }

    const preflightResult = new PreflightResult();
    preflightResult.appSlug = result.rows[0].app_slug;
    preflightResult.clusterSlug = result.rows[0].cluster_slug;
    preflightResult.result = result.rows[0].preflight_result;
    preflightResult.createdAt = result.rows[0].preflight_result_created_at;

    return preflightResult;

  }

  async getLatestKotsPreflightResult(): Promise<PreflightResult> {
    const q = `SELECT id FROM app WHERE current_sequence = 0 ORDER BY created_at DESC LIMIT 1`;
    const r = await this.pool.query(q);
    if (r.rows.length === 0) {
      throw new ReplicatedError(`No app has been installed yet`);
    }
    const appId = r.rows[0].id;

    const qq =
      `SELECT
        app_downstream_version.preflight_result,
        app_downstream_version.preflight_result_created_at,
        app.slug as app_slug,
        cluster.slug as cluster_slug
      FROM app_downstream_version
        INNER JOIN app ON app_downstream_version.app_id = app.id
        INNER JOIN cluster ON app_downstream_version.cluster_id = cluster.id
      WHERE
        app_downstream_version.app_id = $1 AND
        app_downstream_version.sequence = 0`;

    const vv = [ appId ];
    const result = await this.pool.query(qq, vv);
    const preflightResult = new PreflightResult();
    preflightResult.appSlug = result.rows[0].app_slug;
    preflightResult.clusterSlug = result.rows[0].cluster_slug;
    preflightResult.result = result.rows[0].preflight_result;
    preflightResult.createdAt = result.rows[0].preflight_result_created_at;

    return preflightResult;
  }

  async getPendingPreflightParams(inCluster: boolean): Promise<PreflightParams[]> {
    const params = await Params.getParams();
    const q =
      `SELECT
        app_downstream_version.sequence as sequence,
        app_downstream_version.preflight_ignore_permissions,
        app.slug as app_slug,
        cluster.slug as cluster_slug
      FROM app_downstream_version
        INNER JOIN app ON app_downstream_version.app_id = app.id
        INNER JOIN cluster ON app_downstream_version.cluster_id = cluster.id
      WHERE app_downstream_version.status = 'pending_preflight'`;

    const result = await this.pool.query(q);

    const preflightParams: PreflightParams[] = [];
    for (const row of result.rows) {
      const {
        app_slug: appSlug,
        cluster_slug: clusterSlug,
        sequence,
        preflight_ignore_permissions: ignorePermissions,
      } = row;

      let url: string;
      if (inCluster) {
        url = `${params.shipApiEndpoint}/api/v1/preflight/${appSlug}/${clusterSlug}/${sequence}?incluster=true`;
      } else {
        url = `${params.apiAdvertiseEndpoint}/api/v1/preflight/${appSlug}/${clusterSlug}/${sequence}`;
      }
  
      const param: PreflightParams = {
        url: url,
        ignorePermissions: ignorePermissions,
      };
      preflightParams.push(param);
    }

    return preflightParams;
  }

  async getPreflightCommand(appSlug: string, clusterSlug: string, sequence: string): Promise<string> {
    const params = await Params.getParams();
    let url = `${params.apiAdvertiseEndpoint}/api/v1/preflight/${appSlug}/${clusterSlug}/${sequence}`;
    const preflightCommand = `
curl https://krew.sh/preflight | bash
kubectl preflight ${url}
    `;
    return preflightCommand;
  }


}
