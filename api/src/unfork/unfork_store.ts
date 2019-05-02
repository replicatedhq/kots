import * as jaeger from "jaeger-client";
import { instrumented } from "monkit";
import * as randomstring from "randomstring";
import * as rp from "request-promise";
import { Service } from "ts-express-decorators";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import { traced } from "../server/tracing";
import { PostgresWrapper } from "../util/persistence/db";
import { UnforkSession } from "../generated/types";

@Service()
export class UnforkStore {
  constructor(private readonly wrapper: PostgresWrapper, private readonly params: Params) {}

  @instrumented()
  @traced({ paramTags: { userId: 1, upstreamUri: 2, forkUri: 3 } })
  async createUnforkSession(ctx: jaeger.SpanContext, userId: string, upstreamUri: string, forkUri: string): Promise<UnforkSession> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    const q = `INSERT INTO ship_unfork (id, user_id, upstream_uri, fork_uri, created_at)
               VALUES ($1, $2, $3, $4, $5)`;
    const v = [id, userId, upstreamUri, forkUri, new Date()];

    await this.wrapper.query(q, v);

    return this.getSession(ctx, id);
  }

  @instrumented()
  @traced({ paramTags: { unforkSessionId: 1 } })
  async deployUnforkSession(ctx: jaeger.SpanContext, id: string): Promise<UnforkSession> {
    const unforkSession = await this.getSession(ctx, id);

    const options = {
      method: "POST",
      uri: `${this.params.shipInitBaseURL}/v1/unfork`,
      headers: {
        "X-TraceContext": ctx,
      },
      body: {
        id: unforkSession.id,
        upstreamUri: unforkSession.upstreamUri,
        forkUri :unforkSession.forkUri,
      },
      json: true,
    };

    const parsedBody = await rp(options);
    logger.debug({ msg: "response from deploy unfork", parsedBody });

    return unforkSession;
  }

  @instrumented()
  @traced({ paramTags: { unforkSessionId: 1 } })
  async getSession(ctx: jaeger.SpanContext, id: string): Promise<UnforkSession> {
    const q = `
      SELECT id, upstream_uri, fork_uri, created_at, finished_at, result
      FROM ship_unfork
      WHERE id = $1
    `;
    const v = [id];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    return this.mapRow(rows[0]);
  }

  private mapRow(row: any): UnforkSession {
    return {
      id: row.id,
      upstreamUri: row.upstream_uri,
      forkUri: row.fork_uri,
      createdOn: row.created_at,
      finishedOn: row.finished_at,
      result: row.result,
    };
  }
}
