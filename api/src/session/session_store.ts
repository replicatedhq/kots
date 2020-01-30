import { addWeeks } from "date-fns";
import jwt from "jsonwebtoken";
import randomstring from "randomstring";

import { Params } from "../server/params";
import pg from "pg";
import { Session } from "./session";
import { ReplicatedError } from "../server/errors";
import * as k8s from "@kubernetes/client-node";
import {base64Decode} from "../util/utilities";

export type InstallationMap = {
  [key: string]: number;
};

export class SessionStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) { }

  /**
   * Creates a signed JWT for authenticaion
   * @param userId - string of user_id from the database
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
    if (token && token.length > 0 && token !== "null") {
      try {
        if (token.startsWith("Bearer ")) {
          token = token.split(" ")[1];
        }

        if (token.startsWith("Kots ")) {
          // this is a token from the kots CLI
          // it needs to be compared with the "kotsadm-authstring" secret
          // if that matches, we return a new session token with the session ID set to the authstring value
          // and the userID set to "kots-cli"
          // this works for now as the endpoints used by the kots cli don't rely on user ID
          // TODO make real userid/sessionid
          const kc = new k8s.KubeConfig();
          kc.loadFromDefault();
          const k8sApi = kc.makeApiClient(k8s.CoreV1Api);

          const namespace = process.env["POD_NAMESPACE"]!;
          const secretName = "kotsadm-authstring";
          const secret = await k8sApi.readNamespacedSecret(secretName, namespace);
          const data = secret.body.data!;
          const authString = base64Decode(data["kotsadm-authstring"]);
          if (!authString) {
            console.log(`no authstring: ${data}`);
            return new Session();
          } else if (authString !== token) {
            console.log(`authstring from secret is ${authString}, does not match ${token}`);
            return new Session();
          }

          const kotsSession = new Session();
          kotsSession.sessionId = token;
          kotsSession.userId = "kots-cli";
          return kotsSession;
        }

        const decoded: any = jwt.verify(token, this.params.sessionKey);

        const session = await this.getSession(decoded.sessionId);
        return session;
      } catch (e) {
        console.log(e);
        // Errors here negligible as they are from jwts not passing verification
      }
    }

    return new Session();
  }
}
