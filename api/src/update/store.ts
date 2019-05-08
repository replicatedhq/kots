import * as randomstring from "randomstring";
import * as rp from "request-promise";
import { Service } from "ts-express-decorators";
import { UpdateSession } from "../generated/types";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import { PostgresWrapper } from "../util/persistence/db";

@Service()
export class UpdateStore {
  constructor(private readonly wrapper: PostgresWrapper, private readonly params: Params) {}

  async createUpdateSession(userId: string, watchId: string): Promise<UpdateSession> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    const q = `INSERT INTO ship_update (id, user_id, watch_id, created_at)
               VALUES ($1, $2, $3, $4)`;
    const v = [id, userId, watchId, new Date()];

    await this.wrapper.query(q, v);

    const updateSession = this.getSession(id);

    return updateSession;
  }

  async deployUpdateSession(updateSessionId: string): Promise<UpdateSession> {
    const updateSession = await this.getSession(updateSessionId);

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
    const q = `
      SELECT id, watch_id, created_at, finished_at, result
      FROM ship_update
      WHERE id = $1
    `;
    const v = [id];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    const result = this.mapRow(rows[0]);

    return result;
  }

  private mapRow(row: any): UpdateSession {
    return {
      id: row.id,
      watchId: row.watch_id,
      createdOn: row.created_at,
      finishedOn: row.finished_at,
      result: row.result,
    };
  }
}
