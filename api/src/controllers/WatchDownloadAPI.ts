import * as Express from "express";
import { instrumented } from "monkit";
import { Controller, Get, PathParams, Res } from "ts-express-decorators";
import { WatchDownload } from "../watch/download";
import { WatchStore } from "../watch/watch_store";
import * as jaeger from "jaeger-client";
import { tracer } from "../server/tracing";

@Controller("/api/v1/download")
export class WatchDownloadAPI {
  constructor(
    private readonly watchDownload: WatchDownload,
    private readonly watchStore: WatchStore,
  ) {
  }

  @Get("/:watchId")
  @instrumented()
  async downloadDeploymentYAML(@Res() response: Express.Response, @PathParams("watchId") watchId: string): Promise<void> {
    const span: jaeger.SpanContext = tracer().startSpan("watchDownloadAPI.downloadDeploymentYAML");

    const { userId = "" } = response.locals.context || {};
    const watch = await this.watchStore.findUserWatch(span.context(), userId, { id: watchId });

    const { filename, contents, contentType } = await this.watchDownload.downloadDeploymentYAML(watch.id!);

    response.setHeader("Content-Disposition", `attachment; filename=${filename}`);
    response.setHeader("Content-Type", contentType);
    response.send(contents);

    span.finish();
  }

  @Get("/:watchId/:sequence")
  @instrumented()
  async downloadDeploymentYAMLForSequence(@Res() response: Express.Response, @PathParams("watchId") watchId: string, @PathParams("sequence") sequence: number): Promise<void> {
    const span: jaeger.SpanContext = tracer().startSpan("watchDownloadAPI.downloadDeploymentYAML");

    const { userId = "" } = response.locals.context || {};
    const watch = await this.watchStore.findUserWatch(span.context(), userId, { id: watchId });

    const { filename, contents, contentType } = await this.watchDownload.downloadDeploymentYAMLForSequence(watch.id!, sequence);

    response.setHeader("Content-Disposition", `attachment; filename=${filename}`);
    response.setHeader("Content-Type", contentType);
    response.send(contents);

    span.finish();
  }
}
