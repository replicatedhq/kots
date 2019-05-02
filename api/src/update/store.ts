import * as jaeger from "jaeger-client";
import { instrumented } from "monkit";
import * as randomstring from "randomstring";
import * as rp from "request-promise";
import { Service } from "ts-express-decorators";
import { UpdateSession } from "../generated/types";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import { tracer } from "../server/tracing";
import { PostgresWrapper } from "../util/persistence/db";

@Service()
export class UpdateStore {
  constructor(private readonly wrapper: PostgresWrapper, private readonly params: Params) {}

  @instrumented()
  async createUpdateSession(ctx: jaeger.SpanContext, userId: string, watchId: string): Promise<UpdateSession> {
    const span: jaeger.SpanContext = tracer().startSpan("initStore.createUpdateSession", { childOf: ctx });

    const id = randomstring.generate({ capitalization: "lowercase" });

    const q = `INSERT INTO ship_update (id, user_id, watch_id, created_at)
               VALUES ($1, $2, $3, $4)`;
    const v = [id, userId, watchId, new Date()];

    await this.wrapper.query(q, v);

    const updateSession = this.getSession(span.context(), id);

    span.finish();

    return updateSession;
  }

  @instrumented()
  async deployUpdateSession(ctx: jaeger.SpanContext, updateSessionId: string): Promise<UpdateSession> {
    const span: jaeger.SpanContext = tracer().startSpan("initStore.deployUpdateSession", { childOf: ctx });

    const updateSession = await this.getSession(span.context(), updateSessionId);

    const options = {
      method: "POST",
      uri: `${this.params.shipUpdateBaseURL}/v1/update`,
      body: {
        id: updateSession.id,
        watchId: updateSession.watchId,
      },
      json: true,
    };

    const parsedBody = await rp(options);
    logger.debug({
      message: "updateserver-parsedbody",
      parsedBody,
    });
    span.finish();

    return updateSession;
  }

  @instrumented()
  async getSession(ctx: jaeger.SpanContext, id: string): Promise<UpdateSession> {
    const span: jaeger.SpanContext = tracer().startSpan("updateStore.get", {
      childOf: ctx,
    });

    const q = `
      SELECT id, watch_id, created_at, finished_at, result
      FROM ship_update
      WHERE id = $1
    `;
    const v = [id];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    const result = this.mapRow(rows[0]);

    span.finish();

    return result;
  }

  private mapRow(row: any): UpdateSession {
    return {
      id: row.id,
      watchId: row.watch_id,
      createdOn: row.created_at,
      finishedOn: row.finished_at,
      result: row.result,
    };
  }
}
