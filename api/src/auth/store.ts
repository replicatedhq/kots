// @ts-ignore
import { addMinutes } from "date-fns";
import * as jaeger from "jaeger-client";
import { instrumented } from "monkit";
import * as randomstring from "randomstring";
import { Service } from "ts-express-decorators";
import { traced } from "../server/tracing";
import { Connectable, PostgresWrapper } from "../util/persistence/db";
import { GithubNonceModel, UserModel, ScmLeadModel } from "./models";

@Service()
export class UserStore implements Connectable<UserStore> {
  constructor(readonly wrapper: PostgresWrapper) {}

  withWrapper(wrapper: PostgresWrapper) {
    return new UserStore(wrapper);
  }

  async listAllUsers(ctx: jaeger.SpanContext): Promise<UserModel[]> {
    const q = `select github_id from github_user`;
    const v = [];

    const { rows } = await this.wrapper.query(q, v);

    const users: UserModel[] = [];
    for (const row of rows) {
      const user = await this.getUser(ctx, row.github_id);
      if (user.length) {
        users.push(user[0]);
      }
    }

    return users;
  }

  @instrumented({ tags: ["tier:store"] })
  @traced({ paramTags: { githubId: 1 } })
  async getUser(ctx: jaeger.SpanContext, githubId: number): Promise<UserModel[]> {
    const q = `
      SELECT ship_user.id, github_user.username AS github_username, github_user.email AS email
      FROM ship_user
      INNER JOIN github_user ON ship_user.github_id = github_user.github_id
      WHERE github_user.github_id = $1
    `;
    const v = [githubId];

    const { rows } = await this.wrapper.query(q, v);

    return rows as UserModel[];
  }

  @instrumented()
  @traced({ paramTags: { githubId: 1, githubUsername: 2 } })
  async createShipUser(ctx: jaeger.SpanContext, githubId: number, githubUsername: string) {
    const id = randomstring.generate({ capitalization: "lowercase" });

    const uq = "INSERT INTO ship_user (id, github_id, created_at) VALUES ($1, $2, $3)";
    const uv = [id, githubId, new Date()];

    await this.wrapper.query(uq, uv);
  }

  @instrumented()
  @traced({ paramTags: { githubId: 1, githubUsername: 2 } })
  async createGithubUser(ctx: jaeger.SpanContext, githubId: number, githubUsername: string, githubAvatar: string, email?: string | null) {
    let gq = "INSERT INTO github_user (username, github_id, avatar_url) VALUES ($1, $2, $3)";
    let gv = [githubUsername, githubId, githubAvatar];
    if (email) {
      gq = "INSERT INTO github_user (username, github_id, avatar_url, email) VALUES ($1, $2, $3, $4)";
      gv = [...gv, email];
    }

    await this.wrapper.query(gq, gv);
  }

  @instrumented()
  @traced()
  async trackScmLead(ctx: jaeger.SpanContext, preference: string, email: string, provider: string): Promise<ScmLeadModel> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    const currentTime = new Date(Date.now()).toUTCString();
    const q = `
    INSERT INTO
      track_scm_leads (id, deployment_type, email_address, scm_provider, created_at)
    VALUES ($1, $2, $3, $4, $5) RETURNING id
    `;
    const v = [id, preference, email, provider, currentTime];
    const { rows }: { rows: ScmLeadModel[] } = await this.wrapper.query(q, v);
    return rows[0];
  }

  @instrumented()
  @traced({ paramTags: { githubId: 1, email: 2 } })
  async updateGithubUserEmail(ctx: jaeger.SpanContext, githubId: number, email: string) {
    const q = `UPDATE github_user SET email = $2 WHERE github_id = $1`
    const v = [githubId, email];

    await this.wrapper.query(q, v);
  }

  @instrumented()
  @traced({ paramTags: { userId: 1, contributorId: 2 } })
  async saveWatchContributor(ctx: jaeger.SpanContext, userId: String, id: String) {
    const q = "INSERT INTO user_watch (user_id, watch_id) VALUES ($1, $2)";

    const v = [userId, id];

    await this.wrapper.query(q, v);
  }

  @instrumented()
  @traced({ paramTags: { watchId: 1, userId: 2 } })
  async removeExistingWatchContributorsExcept(ctx: jaeger.SpanContext, id: string, userIdToExclude: string) {
    const q = `
    DELETE FROM
      user_watch
    WHERE
      watch_id = $1 AND
      user_id != $2
    `;

    const v = [id, userIdToExclude];

    await this.wrapper.query(q, v);
  }
}

@Service()
export class GithubNonceStore {
  constructor(private readonly wrapper: PostgresWrapper) {}

  @instrumented()
  @traced()
  async createNonce(ctx: jaeger.SpanContext): Promise<GithubNonceModel> {
    const state = randomstring.generate({ capitalization: "lowercase" });

    const currentTime = new Date(Date.now()).toUTCString();
    const q = "INSERT INTO github_nonce (nonce, expire_at) VALUES ($1, $2) RETURNING nonce";
    const v = [state, addMinutes(currentTime, 10)];
    const { rows }: { rows: GithubNonceModel[] } = await this.wrapper.query(q, v);
    return rows[0];
  }

  @instrumented()
  @traced()
  async getNonce(ctx: jaeger.SpanContext, nonce: string): Promise<GithubNonceModel> {
    const q = `SELECT * FROM github_nonce WHERE nonce = $1`;
    const v = [nonce];
    const { rows }: { rows: GithubNonceModel[] } = await this.wrapper.query(q, v);
    return rows[0];
  }

  @instrumented()
  @traced()
  async deleteNonce(ctx: jaeger.SpanContext, nonce: string): Promise<void> {
    const q = `DELETE FROM github_nonce WHERE nonce = $1`;
    const v = [nonce];
    await this.wrapper.query(q, v);
  }
}
