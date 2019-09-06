import pg from "pg";
import randomstring from "randomstring";

import { PreflightSpec, PreflightResult } from "./";
import { ReplicatedError } from "../server/errors";
import { KotsPreflightResult } from "./preflight";

export class PreflightStore {
  constructor(
    private readonly pool: pg.Pool
  ) {}

  private mapPreflightResult(row: any): PreflightResult {
    const preflightResult = new PreflightResult();
    preflightResult.watchId = row.watch_id;
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

  async getKotsPreflightResult(appId: string, clusterId: string, sequence: number): Promise<KotsPreflightResult> {
    const q = `
      SELECT preflight_result, preflight_result_updated_at, cluster_id
        FROM app_downstream_version
        WHERE app_id = $1 AND cluster_id = $2 AND sequence = $3
    `;

    const v = [
      appId,
      clusterId,
      sequence,
    ];

    const result = await this.pool.query(q,v);

    if (result.rows.length === 0) {
      throw new Error(`Couldn't find preflight spec with appId: ${appId}, clusterId: ${clusterId}, sequence: ${sequence}`);
    }

    const kotsPreflightResult = new KotsPreflightResult();
    kotsPreflightResult.result = result.rows[0].preflight_result;
    kotsPreflightResult.updatedAt = result.rows[0].preflight_result_updated_at;
    kotsPreflightResult.clusterId = result.rows[0].cluster_id;

    return kotsPreflightResult;

  }

  async getLatestKotsPreflightResult(): Promise<KotsPreflightResult> {
    const q = `SELECT id FROM app WHERE current_sequence = 0 ORDER BY created_at DESC LIMIT 1`;
    const r = await this.pool.query(q);
    const appId = r.rows[0].id;

    const qq =
      `SELECT preflight_result, preflight_result_updated_at, cluster_id, app_id FROM app_downstream_version WHERE sequence = 0 AND app_id = $1`;

    const vv = [ appId ];

    const result = await this.pool.query(qq,vv);
    const kotsPreflightResult = new KotsPreflightResult();
    kotsPreflightResult.appId = result.rows[0].app_id;
    kotsPreflightResult.result = result.rows[0].preflight_result;
    kotsPreflightResult.updatedAt = result.rows[0].preflight_result_updated_at;
    kotsPreflightResult.clusterId = result.rows[0].cluster_id;

    return kotsPreflightResult;


  }
}
