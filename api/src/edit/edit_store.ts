import { Params } from "../server/params";
import * as pg from "pg";
import * as rp from "request-promise";
import { EditSession } from "./edit_session";
import * as randomstring from "randomstring";
import { logger } from "../server/logger";

export class EditStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) {}

  async createEditSession(userId: string, watchId: string): Promise<EditSession> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    const q = `insert into ship_edit (id, user_id, watch_id, created_at) values ($1, $2, $3, $4)`;
    const v = [
      id,
      userId,
      watchId,
      new Date(),
    ];

    await this.pool.query(q, v);

    return this.getSession(id);
  }

  async deployEditSession(id: string): Promise<EditSession> {
    const editSession = await this.getSession(id);

    const options = {
      method: "POST",
      uri: `${this.params.shipEditBaseURL}/v1/edit`,
      body: {
        id: editSession.id,
        watchId: editSession.watchId,
      },
      json: true,
    };

    const parsedBody = await rp(options);
    logger.debug({
      message: "editserver-parsedbody",
      parsedBody,
    });

    return editSession;
  }


  async getSession(id: string): Promise<EditSession> {
    const q = `select id, watch_id, created_at, finished_at, result from ship_edit where id = $1`;
    const v = [id];

    const result = await this.pool.query(q, v);
    const session: EditSession = {
      id: result.rows[0].id,
      watchId: result.rows[0].watch_id,
      createdOn: new Date(result.rows[0].created_at),
      finishedOn: result.rows[0].finished_at ? new Date(result.rows[0].finished_at) : undefined,
      result: result.rows[0].result,
    }

    return session;
  }
}
