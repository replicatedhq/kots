import * as GitHubApi from "@octokit/rest";
import { addWeeks } from "date-fns";
// @ts-ignore
import * as jaeger from "jaeger-client";
import * as jwt from "jsonwebtoken";
import { instrumented } from "monkit";
import * as randomstring from "randomstring";
import { Service } from "ts-express-decorators";

import { Params } from "../server/params";
import { traced } from "../server/tracing";
import { PostgresWrapper } from "../util/persistence/db";
import { SessionModel } from "./models";

export type InstallationMap = {
  [key: string]: number;
};

@Service()
export class SessionStore {
  constructor(private readonly wrapper: PostgresWrapper, private readonly params: Params) {}

  createInstallationMap(installations: GitHubApi.GetInstallationsResponseInstallationsItem[]): InstallationMap {
    return installations.reduce((installationAcctMap: InstallationMap, { id, account }) => {
      const lowerLogin = account.login.toLowerCase();
      installationAcctMap[lowerLogin] = id;

      return installationAcctMap;
    }, {});
  }

  @instrumented()
  @traced({ paramTags: { userId: 2 } })
  async createGithubSession(ctx: jaeger.SpanContext, token: string, userId: string): Promise<string> {
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

    const installationMap = this.createInstallationMap(installations);

    const sessionId = randomstring.generate({ capitalization: "lowercase" });

    const q = `INSERT INTO session (id, user_id, metadata, expire_at) VALUES($1, $2, $3, $4)`;

    const currentUtcDate = new Date(Date.now()).toUTCString();
    const expirationDate = addWeeks(currentUtcDate, 2);
    const v = [sessionId, userId, JSON.stringify(installationMap), expirationDate];

    await this.wrapper.query(q, v);

    return jwt.sign(
      {
        token,
        sessionId,
      },
      this.params.sessionKey,
    );
  }

  @instrumented()
  @traced()
  async refreshGithubTokenMetadata(ctx: jaeger.SpanContext, token: string, sessionId: string): Promise<void> {
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

    const q = `UPDATE session SET metadata = $1 WHERE id = $2`;
    const v = [updatedInstallationMap, sessionId];
    await this.wrapper.query(q, v);
  }

  @instrumented()
  @traced()
  async getGithubSession(ctx: jaeger.SpanContext, sessionId: string): Promise<SessionModel> {
    const q = `SELECT id, user_id, metadata, expire_at FROM session WHERE id = $1`;
    const v = [sessionId];
    const { rows }: { rows: SessionModel[] } = await this.wrapper.query(q, v);
    return rows[0];
  }

  @instrumented()
  @traced()
  async deleteSession(ctx: jaeger.SpanContext, sessionId: string): Promise<void> {
    const q = `DELETE FROM session WHERE id = $1`;
    const v = [sessionId];

    await this.wrapper.query(q, v);
  }
}
