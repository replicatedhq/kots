import { Params } from "../server/params";
import pg from "pg";
import rp from "request-promise";
import { EditSession } from "./edit_session";
import randomstring from "randomstring";
import { logger } from "../server/logger";

export class EditStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) {}

  async createEditSession(userId: string, watchId: string, isHeadless: boolean): Promise<EditSession> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    const q = `insert into ship_edit (id, user_id, watch_id, created_at, is_headless) values ($1, $2, $3, $4, $5)`;
    const v = [
      id,
      userId,
      watchId,
      new Date(),
      isHeadless
    ];

    await this.pool.query(q, v);

    return this.getSession(id);
  }

  async deployEditSession(id: string): Promise<EditSession> {
    const editSession = await this.getSession(id);

    const options: any = {
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
      msg: "received parsed body from edit server",
      parsedBody,
    });

    return editSession;
  }


  async getSession(id: string): Promise<EditSession> {
    const q = `select id, watch_id, created_at, finished_at, result, is_headless from ship_edit where id = $1`;
    const v = [id];

    const result = await this.pool.query(q, v);
    const session: EditSession = {
      id: result.rows[0].id,
      watchId: result.rows[0].watch_id,
      createdOn: new Date(result.rows[0].created_at),
      finishedOn: result.rows[0].finished_at ? new Date(result.rows[0].finished_at) : undefined,
      result: result.rows[0].result,
      isHeadless: result.rows[0].is_headless ? result.rows[0].is_headless : false,
    };

    return session;
  }
}
