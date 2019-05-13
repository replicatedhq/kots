import * as randomstring from "randomstring";
import * as rp from "request-promise";
import { UpdateSession } from "./";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import * as pg from "pg";

export class UpdateStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) {}

  async createUpdateSession(userId: string, watchId: string): Promise<UpdateSession> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    const q = `INSERT INTO ship_update (id, user_id, watch_id, created_at)
              VALUES ($1, $2, $3, $4)`;
    const v = [id, userId, watchId, new Date()];

    await this.pool.query(q, v);

    const updateSession = this.getSession(id, userId);

    return updateSession;
  }

  async deployUpdateSession(updateSessionId: string, userId: string): Promise<UpdateSession> {
    const updateSession = await this.getSession(updateSessionId, userId);

    const options = {
      method: "POST",
      uri: `${this.params.shipUpdateBaseURL}/v1/update`,
      body: {
        id: updateSession.id,
        watchId: updateSession.watchId,
      },
      json: true,
    };

    const parsedBody = await rp(options);
    logger.debug({
      message: "updateserver-parsedbody",
      parsedBody,
    });

    return updateSession;
  }

  async getSession(id: string, userId: string): Promise<UpdateSession> {
    const q = `
      SELECT id, watch_id, created_at, finished_at, result
      FROM ship_update
      WHERE id = $1
    `;
    const v = [id];

    const result = await this.pool.query(q, v);
    const session = this.mapRow(result.rows[0], userId);

    return session;
  }

  private mapRow(row: any, userId: string): UpdateSession {
    return {
      id: row.id,
      watchId: row.watch_id,
      createdOn: row.created_at,
      finishedOn: row.finished_at,
      result: row.result,
      userId
    };
  }
}
