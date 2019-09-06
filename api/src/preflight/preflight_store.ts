import pg from "pg";
import randomstring from "randomstring";

import { PreflightSpec, PreflightResult } from "./";
import { ReplicatedError } from "../server/errors";

export class PreflightStore {
  constructor(
    private readonly pool: pg.Pool
  ) {}

  private mapPreflightResult(row: any): PreflightResult {
    const preflightResult = new PreflightResult();
    preflightResult.appId = row.watch_id;
    preflightResult.result = row.result;
    preflightResult.createdAt = row.created_at;

    return preflightResult;
  }

  private mapPreflightSpec(row: any): PreflightSpec {
    const preflightSpec = new PreflightSpec();
    preflightSpec.spec = row.spec;

    return preflightSpec;
  }

  async getPreflightsResultsByWatchId(watchId: string): Promise<PreflightResult[]> {
    const q =
      `SELECT result, created_at, watch_id FROM preflight_result WHERE watch_id = $1 ORDER BY created_at DESC`;
    const v = [ watchId ];

    const result = await this.pool.query(q, v);
    const preflightResults = result.rows.map(this.mapPreflightResult);

    return preflightResults;
  }

  async getPreflightsResultsBySlug(slug: string): Promise<PreflightResult[]> {
    const findWatchBySlugQuery =
      `SELECT id FROM watch WHERE slug = $1`;

    const watchIdBySlugResult = await this.pool.query(findWatchBySlugQuery, [ slug ]);

    const id = watchIdBySlugResult.rows[0].id;

    return await this.getPreflightsResultsByWatchId(id);

  }

  async getPreflightSpec(watchId: string): Promise<PreflightSpec> {
    const q =
      `SELECT spec FROM preflight_spec WHERE watch_id = $1 ORDER BY sequence DESC LIMIT 1`;
    const v = [ watchId ];

    const result = await this.pool.query(q, v);

    if (!result.rows[0]) {
      throw new ReplicatedError(`Couldn't find PreflightSpec for watchId: ${watchId}`);
    }

    const spec = this.mapPreflightSpec(result.rows[0]);

    return spec;
  }

  async getPreflightSpecBySlug(slug: string): Promise<PreflightSpec> {
    const q =
      `SELECT preflight_spec.spec FROM watch
          LEFT JOIN preflight_spec ON preflight_spec.watch_id = watch.id
          INNER JOIN watch_version ON watch_version.watch_id = watch.id
            AND watch_version.sequence = preflight_spec.sequence
       WHERE watch.slug = $1
       ORDER BY watch_version.sequence DESC LIMIT 1`;
    const v = [ slug ];
    const results = await this.pool.query(q, v);

    if (results.rowCount === 0) {
      throw new ReplicatedError(`Couldn't find PreflightSpec for slug: ${slug}`);
    }
    const spec = this.mapPreflightSpec(results.rows[0]);
    return spec;
  }

  async addPreflightResult(slug: string, result: string): Promise<void> {
    const watchIdFromSlugQuery = `SELECT id FROM watch WHERE slug = $1`;
    const watchIdFromSlugResults = await this.pool.query(watchIdFromSlugQuery, [ slug ]);

    const watchId = watchIdFromSlugResults.rows[0].id;

    const q =
      `INSERT INTO preflight_result (id, watch_id, result, created_at) VALUES ($1, $2, $3, NOW())`;
    const id = randomstring.generate({ capitalization: "lowercase" });

    const v = [
      id,
      watchId,
      result,
    ];

    await this.pool.query(q,v);
  }

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
        app_downstream_version.preflight_result_updated_at,
        app_downstream_version.cluster_id,
        app.id as app_id,
        app.slug as app_slug
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
    preflightResult.appId = result.rows[0].app_id;
    preflightResult.appSlug = result.rows[0].app_slug;
    preflightResult.result = result.rows[0].preflight_result;
    preflightResult.createdAt = result.rows[0].preflight_result_updated_at;

    return preflightResult;

  }

  async getLatestKotsPreflightResult(): Promise<PreflightResult> {
    const q = `SELECT id FROM app WHERE current_sequence = 0 ORDER BY created_at DESC LIMIT 1`;
    const r = await this.pool.query(q);
    const appId = r.rows[0].id;

    const qq =
      `SELECT
        app_downstream_version.preflight_result,
        app_downstream_version.preflight_result_updated_at,
        app_downstream_version.cluster_id,
        app.id as app_id,
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
    preflightResult.appId = result.rows[0].app_id;
    preflightResult.appSlug = result.rows[0].app_slug;
    preflightResult.clusterId = result.rows[0].cluster_id;
    preflightResult.clusterSlug = result.rows[0].cluster_slug;
    preflightResult.result = result.rows[0].preflight_result;
    preflightResult.createdAt = result.rows[0].preflight_result_updated_at;

    return preflightResult;
  }
}
