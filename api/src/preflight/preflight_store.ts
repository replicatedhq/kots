import pg from "pg";

import { Params } from "../server/params";
import { PreflightResult } from "./";
import { ReplicatedError } from "../server/errors";

export class PreflightStore {
  constructor(
    private readonly pool: pg.Pool
  ) {}

  async getKotsPreflightSpec(appId: string, sequence: number): Promise<string> {
    const q = `
      SELECT preflight_spec FROM app_version WHERE app_id = $1 AND sequence = $2
    `;

    const result = await this.pool.query(q, [appId, sequence]);

    if (result.rows.length === 0) {
      throw new ReplicatedError(`Unable to find Preflight Spec with appId ${appId}`);
    }

    return result.rows[0].preflight_spec;
  }

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

  async getPendingPreflightUrls(): Promise<string[]> {
    const params = await Params.getParams();
    const q =
      `SELECT
        app_downstream_version.sequence as sequence,
        app.slug as app_slug,
        cluster.slug as cluster_slug
      FROM app_downstream_version
        INNER JOIN app ON app_downstream_version.app_id = app.id
        INNER JOIN cluster ON app_downstream_version.cluster_id = cluster.id
      WHERE app_downstream_version.status = 'pending_preflight'`;

    const result = await this.pool.query(q);

    const preflightUrls: string[] = [];
    for (const row of result.rows) {
      const {
        app_slug: appSlug,
        cluster_slug: clusterSlug,
        sequence
      } = row;
      preflightUrls.push(`${params.shipApiEndpoint}/api/v1/preflight/${appSlug}/${clusterSlug}/${sequence}`);
    }

    return preflightUrls;
  }


}
