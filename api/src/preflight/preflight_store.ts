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
      `SELECT spec FROM preflight_spec WHERE watch_id = $1 ORDER BY watch_sequence DESC LIMIT 1`;
    const v = [ watchId ];

    const result = await this.pool.query(q, v);

    if (!result.rows[0]) {
      throw new ReplicatedError(`Couldn't find PreflightSpec for watchId: ${watchId}`);
    }

    const spec = this.mapPreflightSpec(result.rows[0]);

    return spec;
  }

  async getPreflightSpecBySlug(slug: string): Promise<PreflightSpec> {
    const q = `SELECT spec FROM preflight_spec LEFT OUTER JOIN watch ON preflight_spec.watch_id = watch.id WHERE watch.slug = $1`;
    const v = [ slug ];
    const results = await this.pool.query(q, v);

    if (!results.rows[0]) {
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
}
