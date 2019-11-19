import { addWeeks } from "date-fns";
import jwt from "jsonwebtoken";
import randomstring from "randomstring";

import { Params } from "../server/params";
import pg from "pg";
import { Session } from "./session";
import { ReplicatedError } from "../server/errors";

export type InstallationMap = {
  [key: string]: number;
};

export class SessionStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) {}

  /**
   * Creates a signed JWT for authenticaion
   * @param userId - string of user_id from the database
   * @param isSingleTenant - true if the user is coming from secure admin console. See user_mutations.ts#loginToAdminConsole
   */
  async createPasswordSession(userId: string): Promise<string> {
    const sessionId = randomstring.generate({ capitalization: "lowercase" });
    const currentUtcDate = new Date(Date.now()).toUTCString();
    const expirationDate = addWeeks(currentUtcDate, 2);

    const q = `insert into session (id, user_id, metadata, expire_at) values ($1, $2, $3, $4)`;
    const v = [
      sessionId,
      userId,
      "{}",
      expirationDate,
    ];

    await this.pool.query(q, v);

    return jwt.sign(
      {
        type: "ship",
        sessionId,
        isSingleTenant: true,
      },
      this.params.sessionKey
    );
  }

  public async deleteSession(sessionId: string): Promise<void> {
    const q = `delete from session where id = $1`;
    const v = [sessionId];

    await this.pool.query(q, v);
  }

  public async getSession(id: string): Promise<Session> {
    const q = `select id, user_id, expire_at from session where id = $1`;
    const v = [id];

    const result = await this.pool.query(q, v);

    const session: Session = new Session();

    if (!result.rows.length) {
      throw new ReplicatedError(`Session not found with ID ${id}`);
    }

    session.type = "ship";
    session.sessionId = result.rows[0].id;
    session.userId = result.rows[0].user_id;
    session.expiresAt = result.rows[0].expire_at;

    return session;
  }

  public async decode(token: string): Promise<Session> {
    //tslint:disable-next-line possible-timing-attack
    if (token.length > 0 && token !== "null") {
      try {
        const decoded: any = jwt.verify(token, this.params.sessionKey);

        const session = await this.getSession(decoded.sessionId);
        return session;
      } catch (e) {
        // Errors here negligible as they are from jwts not passing verification
      }
    }

    return new Session();
  }
}
