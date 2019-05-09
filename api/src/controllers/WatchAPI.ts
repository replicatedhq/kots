import * as Express from "express";
import { Controller, Get, Res, Req, QueryParams, PathParams } from "ts-express-decorators";
import * as jaeger from "jaeger-client";
import { tracer } from "../server/tracing";
import * as _ from "lodash";

interface ErrorResponse {
  error: {};
}

@Controller("/api/v1/watch")
export class WatchAPI {
  @Get("/:upstreamId/upstream.yaml")
  async getUpstreamWatch(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
    @PathParams("upstreamId") upstreamId: string,
    @QueryParams("token") token: string,
  ): Promise<any | ErrorResponse> {
    const span: jaeger.SpanContext = tracer().startSpan("watchApi.getUpstreamWatch");

    const watch = await request.app.locals.stores.watchStore.findUpstreamWatch(span.context(), token, upstreamId);

    const params = await request.app.locals.stores.watchStore.getLatestGeneratedFileS3Params(span, watch.id!);

    const download = await request.app.locals.stores.downloadService.findDeploymentFile(span.context(), params);

    span.finish();
    response.status(200);

    return download.contents.toString("utf-8");
  }
}
