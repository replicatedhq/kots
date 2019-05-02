import * as Express from "express";
import { Controller, Get, Res, QueryParams, PathParams } from "ts-express-decorators";
import * as jaeger from "jaeger-client";
import { tracer } from "../server/tracing";
import { WatchStore } from "../watch/watch_store";
import { WatchDownload } from "../watch/download";
import * as _ from "lodash";

interface ErrorResponse {
  error: {};
}

@Controller("/api/v1/watch")
export class WatchAPI {
  constructor(
    private readonly watchStore: WatchStore,
    private readonly downloadService: WatchDownload,
  ) {
  }

  @Get("/:upstreamId/upstream.yaml")
  async getUpstreamWatch(
    @Res() response: Express.Response,
    @PathParams("upstreamId") upstreamId: string,
    @QueryParams("token") token: string,
  ): Promise<any | ErrorResponse> {
    const span: jaeger.SpanContext = tracer().startSpan("watchApi.getUpstreamWatch");

    const watch = await this.watchStore.findUpstreamWatch(span.context(), token, upstreamId);

    const params = await this.watchStore.getLatestGeneratedFileS3Params(span, watch.id!);

    const download = await this.downloadService.findDeploymentFile(span.context(), params);

    span.finish();
    response.status(200);

    return download.contents.toString("utf-8");
  }
}
