import { PendingInitSession } from "./";
import { Params } from "../server/params";
import * as pg from "pg";

export class PendingStore {
  constructor(
    private readonly pool: pg.Pool,
    private readonly params: Params
  ) {
  }

  public async listPendingInitSessions(userId: string): Promise<PendingInitSession[]> {
    const q = `select id, title, upstream_uri, requested_upstream_uri, created_at, finished_at, result from ship_init_pending
      inner join ship_initPending_user on ship_init_pending_id = ship_init_pending.id where
      where user_id = $1`;
    const v = [
      userId,
    ];

    const result = await this.pool.query(q, v);
    const pendingInitSessions: PendingInitSession[] = [];
    for (const row of result.rows) {
      const pendingInitSession: PendingInitSession = {
        id: row.id,
        title: row.title,
        upstreamURI: row.upstream_uri,
        requestedUpstreamURI: row.requested_upstream_uri,
        createdAt: new Date(row.created_at),
        finishedAt: row.finished_at ? new Date(row.finished_at) : undefined,
        result: row.result,
      };

      pendingInitSessions.push(pendingInitSession);
    }

    return pendingInitSessions;
  }
}
