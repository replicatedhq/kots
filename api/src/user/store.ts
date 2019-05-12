// @ts-ignore
import { addMinutes } from "date-fns";
import * as jaeger from "jaeger-client";
import * as randomstring from "randomstring";
import * as pg from "pg";
import { GithubNonce, ScmLead } from "./";

export class UserStoreOld {
  constructor(readonly pool: pg.Pool) {}

  async saveWatchContributor(ctx: jaeger.SpanContext, userId: String, id: String) {
    const q = "INSERT INTO user_watch (user_id, watch_id) VALUES ($1, $2)";

    const v = [userId, id];

    await this.pool.query(q, v);
  }

  async removeExistingWatchContributorsExcept(ctx: jaeger.SpanContext, id: string, userIdToExclude: string) {
    const q = `
    DELETE FROM
      user_watch
    WHERE
      watch_id = $1 AND
      user_id != $2
    `;

    const v = [id, userIdToExclude];

    await this.pool.query(q, v);
  }
}

export class GithubNonceStore {
  constructor(private readonly pool: pg.Pool) {}

  async createNonce(): Promise<GithubNonce> {
    const state = randomstring.generate({ capitalization: "lowercase" });

    const currentTime = new Date(Date.now()).toUTCString();

    const q = "INSERT INTO github_nonce (nonce, expire_at) VALUES ($1, $2) RETURNING nonce";
    const v = [state, addMinutes(currentTime, 10)];
    const { rows }: { rows: GithubNonce[] } = await this.pool.query(q, v);
    return rows[0];
  }

  async getNonce(nonce: string): Promise<GithubNonce> {
    const q = `SELECT * FROM github_nonce WHERE nonce = $1`;
    const v = [nonce];
    const { rows }: { rows: GithubNonce[] } = await this.pool.query(q, v);
    return rows[0];
  }

  async deleteNonce(nonce: string): Promise<void> {
    const q = `DELETE FROM github_nonce WHERE nonce = $1`;
    const v = [nonce];
    await this.pool.query(q, v);
  }
}
