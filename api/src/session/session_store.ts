import * as GitHubApi from "@octokit/rest";
import { addWeeks } from "date-fns";
import * as jwt from "jsonwebtoken";
import * as randomstring from "randomstring";

import { Params } from "../server/params";
import * as pg from "pg";
import { Session } from "./session";

const invalidSession = {
  auth: "",
  expiration: new Date(Date.now()),
  userId: "",
  sessionId: "",
  type: "",
};

export type InstallationMap = {
  [key: string]: number;
};

export class SessionStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) {}

  createInstallationMap(installations: GitHubApi.GetInstallationsResponseInstallationsItem[]): InstallationMap {
    return installations.reduce((installationAcctMap: InstallationMap, { id, account }) => {
      const lowerLogin = account.login.toLowerCase();
      installationAcctMap[lowerLogin] = id;

      return installationAcctMap;
    }, {});
  }

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
      },
      this.params.sessionKey
    );
  }

  async createGithubSession(userId: string, github: any, token: string): Promise<string> {
    const { data: installationData } = await github.users.getInstallations({});
    const { installations } = installationData as {
      total_count: number;
      installations: GitHubApi.GetInstallationsResponseInstallationsItem[];
    };

    const installationMap = this.createInstallationMap(installations);

    const sessionId = randomstring.generate({ capitalization: "lowercase" });
    const currentUtcDate = new Date(Date.now()).toUTCString();
    const expirationDate = addWeeks(currentUtcDate, 2);

    const q = `insert into session (id, user_id, metadata, expire_at) values ($1, $2, $3, $4)`;
    const v = [
      sessionId,
      userId,
      JSON.stringify(installationMap),
      expirationDate
    ];

    await this.pool.query(q, v);

    return jwt.sign(
      {
        type: "github",
        token,
        sessionId,
      },
      this.params.sessionKey,
    );
  }

  async refreshGithubTokenMetadata(token: string, sessionId: string): Promise<void> {
    const github = new GitHubApi();
    github.authenticate({
      type: "token",
      token,
    });
    const { data: installationData } = await github.users.getInstallations({});
    const { installations } = installationData as {
      total_count: number;
      installations: GitHubApi.GetInstallationsResponseInstallationsItem[];
    };

    const updatedInstallationMap = this.createInstallationMap(installations);

    const q = `update session set metadata = $1 where id = $2`;
    const v = [updatedInstallationMap, sessionId];
    await this.pool.query(q, v);
  }

  public async getGithubSession(sessionId: string): Promise<Session> {
    const q = `select id, user_id, metadata, expire_at from session where id = $1`;
    const v = [sessionId];

    const result = await this.pool.query(q, v);

    const session: Session = new Session();
    session.sessionId = result.rows[0].id;
    session.userId = result.rows[0].user_id;
    session.metadata = result.rows[0].metadata;
    session.expiresAt = result.rows[0].expire_at;

    return session;
  }

  public async deleteSession(sessionId: string): Promise<void> {
    const q = `delete from session where id = $1`;
    const v = [sessionId];

    await this.pool.query(q, v);
  }

  public async decode(token: string): Promise<Session | any> {
    //tslint:disable-next-line possible-timing-attack
    if (token.length > 0 && token !== "null") {
      try {
        const decoded: any = jwt.verify(token, this.params.sessionKey);

        if (decoded.type === "github") {
          await this.refreshGithubTokenMetadata(decoded.token, decoded.sessionId);
        }

        const session = await this.getGithubSession(decoded.sessionId);
        return {
          scmToken: decoded.token,
          metadata: JSON.parse(session.metadata),
          expiration: session.expiresAt,
          userId: session.userId,
          sessionId: session.sessionId,
          type: decoded.type,
        };
      } catch (e) {
        // Errors here negligible as they are from jwts not passing verification
      }
    }

    return invalidSession;
  }
}
