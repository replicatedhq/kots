import * as randomstring from "randomstring";
import * as rp from "request-promise";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import * as pg from "pg";
import { UnforkSession } from "./";

export class UnforkStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) {}

  async createUnforkSession(userId: string, upstreamUri: string, forkUri: string): Promise<UnforkSession> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    const q = `INSERT INTO ship_unfork (id, user_id, upstream_uri, fork_uri, created_at)
              VALUES ($1, $2, $3, $4, $5)`;
    const v = [id, userId, upstreamUri, forkUri, new Date()];

    await this.pool.query(q, v);

    return this.getSession(id);
  }

  async deployUnforkSession(id: string): Promise<UnforkSession> {
    const unforkSession = await this.getSession(id);

    const options = {
      method: "POST",
      uri: `${this.params.shipInitBaseURL}/v1/unfork`,
      body: {
        id: unforkSession.id,
        upstreamUri: unforkSession.upstreamURI,
        forkUri :unforkSession.forkURI,
      },
      json: true,
    };

    const parsedBody = await rp(options);
    logger.debug({msg: "response from deploy unfork", parsedBody});

    return unforkSession;
  }

  async getSession(id: string): Promise<UnforkSession> {
    const q = `
      SELECT id, upstream_uri, fork_uri, created_at, finished_at, result
      FROM ship_unfork
      WHERE id = $1
    `;
    const v = [id];

    const result = await this.pool.query(q, v);
    return this.mapRow(result.rows[0]);
  }

  private mapRow(row: any): UnforkSession {
    return {
      id: row.id,
      upstreamURI: row.upstream_uri,
      forkURI: row.fork_uri,
      createdOn: new Date(row.created_at),
      finishedOn: row.finished_at ? new Date(row.finished_at) : undefined,
      result: row.result,
    };
  }
}
