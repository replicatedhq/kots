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

    const q = `insert into ship_update (id, user_id, watch_id, created_at) values ($1, $2, $3, $4)`;
    const v = [
      id,
      userId,
      watchId,
      new Date(),
    ];

    await this.pool.query(q, v);

    return this.getSession(id);
  }

  async deployUpdateSession(id: string): Promise<UpdateSession> {
    const updateSession = await this.getSession(id);

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

  async getSession(id: string): Promise<UpdateSession> {
    const q = `select id, watch_id, created_at, finished_at, result from ship_update where id = $1`;
    const v = [id];

    const result = await this.pool.query(q, v);
    const session: UpdateSession = {
      id: result.rows[0].id,
      watchId: result.rows[0].watch_id,
      createdOn: new Date(result.rows[0].created_at),
      finishedOn: result.rows[0].finished_at ? new Date(result.rows[0].finished_at) : undefined,
      result: result.rows[0].result,
    }

    return session;
  }
}
