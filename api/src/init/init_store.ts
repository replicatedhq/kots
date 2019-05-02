import * as jaeger from "jaeger-client";
import { instrumented } from "monkit";
import * as randomstring from "randomstring";
import * as rp from "request-promise";
import { Service } from "ts-express-decorators";
import { InitSession } from "../generated/types";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import { traced } from "../server/tracing";
import { PostgresWrapper } from "../util/persistence/db";

@Service()
export class InitStore {
  constructor(
    private readonly wrapper: PostgresWrapper,
    private readonly params: Params
  ) {
  }

  @instrumented()
  @traced({ paramTags: { userId: 1, upstream: 2 } })
  async createInitSession(ctx: jaeger.SpanContext, userId: string, upstreamUri: string, clusterId: any, githubPath: any, requestedUpstreamUri?: string): Promise<InitSession> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    const q = `insert into ship_init (id, user_id, upstream_uri, created_at, cluster_id, github_path, requested_upstream_uri) values ($1, $2, $3, $4, $5, $6, $7)`;
    const v = [
      id,
      userId,
      upstreamUri,
      new Date(),
      clusterId,
      githubPath,
      requestedUpstreamUri,
    ];

    await this.wrapper.query(q, v);

    return this.getSession(ctx, id);
  }

  @instrumented()
  @traced({ paramTags: { initSessionId: 1 } })
  async deployInitSession(ctx: jaeger.SpanContext, id: string): Promise<InitSession> {
    const initSession = await this.getSession(ctx, id);

    if (this.params.skipDeployToWorker) {
      logger.info("skipping deploy to worker");
      return initSession;
    }

    const options = {
      method: "POST",
      uri: `${this.params.shipInitBaseURL}/v1/init`,
      headers: {
        "X-TraceContext": ctx,
      },
      body: {
        id: initSession.id,
        upstreamUri: initSession.upstreamUri,
      },
      json: true,
    };

    const parsedBody = await rp(options);
    logger.debug({ msg: "response from deploy init", parsedBody });
    return initSession;
  }

  @instrumented()
  @traced({ paramTags: { initSessionId: 1 } })
  async getSession(ctx: jaeger.SpanContext, id: string): Promise<InitSession> {
    const q = `
      SELECT id, upstream_uri, created_at, finished_at, result
      FROM ship_init
      WHERE id = $1
    `;
    const v = [id];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    return this.mapRow(rows[0]);
  }

  private mapRow(row: any): InitSession {
    return {
      id: row.id,
      upstreamUri: row.upstream_uri,
      createdOn: row.created_at,
      finishedOn: row.finished_at,
      result: row.result,
    };
  }
}
