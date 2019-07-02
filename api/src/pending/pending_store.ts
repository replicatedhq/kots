import { PendingInitSession } from "./";
import { Params } from "../server/params";
import { ReplicatedError } from "../server/errors";
import * as pg from "pg";

export class PendingStore {
  constructor(
    private readonly pool: pg.Pool,
    private readonly params: Params
  ) {
  }

  public async listPendingInitSessions(userId: string): Promise<PendingInitSession[]> {
    const q = `select id, title, upstream_uri, requested_upstream_uri, created_at, finished_at, result from ship_init_pending
      inner join ship_init_pending_user on ship_init_pending_id = ship_init_pending.id
      where user_id = $1`;
    const v = [
      userId,
    ];

    const result = await this.pool.query(q, v);
    const pendingInitSessions: PendingInitSession[] = [];
    for (const row of result.rows) {
      pendingInitSessions.push(this.mapRow(row));
    }

    return pendingInitSessions;
  }

  public async getPendingInitURI(initId: string): Promise<string> {
    const q = `select upstream_uri from ship_init_pending where id = $1`
    const v = [initId];
    const result = await this.pool.query(q,v);

    return result.rows[0].upstream_uri;
  }

  public async getPendingInitSession(initId: string, userId: string): Promise<PendingInitSession> {
    const q = `select id, title, upstream_uri, requested_upstream_uri, created_at, finished_at, result from ship_init_pending
    inner join ship_init_pending_user on ship_init_pending_id = ship_init_pending.id
    where id = $1 and user_id = $2`;
    const v = [initId, userId];

    const result = await this.pool.query(q, v);

    if (result.rows.length === 0) {
      throw new ReplicatedError(`No pending init found for ID ${initId}. Check your watch dashboard to see if the installation was completed`);
    }

    return result.rows[0];
  }

  public async searchPendingInitSessions(userId: string, title: string): Promise<PendingInitSession[]> {
      const q = `
      select
        id, title, upstream_uri, requested_upstream_uri, created_at, finished_at, result
      from
        ship_init_pending
      inner join
        ship_init_pending_user on ship_init_pending_id = ship_init_pending.id
      where
        user_id = $1
      and
        title ILIKE $2`;

      const v = [
        userId,
        `%${title}%`,
      ];

      const result = await this.pool.query(q, v);
      const pendingInitSessions: PendingInitSession[] = [];
      for (const row of result.rows) {
        const result = this.mapRow(row);
        pendingInitSessions.push(result);
      }
      return pendingInitSessions;
  }

  private mapRow(row): PendingInitSession {
    return {
      id: row.id,
      title: row.title,
      upstreamURI: row.upstream_uri,
      requestedUpstreamURI: row.requested_upstream_uri,
      createdAt: new Date(row.created_at),
      finishedAt: row.finished_at ? new Date(row.finished_at) : undefined,
      result: row.result,
    }
  }

}
