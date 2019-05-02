import * as Express from "express";
import { Controller, Get, Res, HeaderParams } from "ts-express-decorators";
import { ClusterStore } from "../cluster/cluster_store";
import * as BasicAuth from "basic-auth";
import * as jaeger from "jaeger-client";
import { tracer } from "../server/tracing";
import { WatchStore } from "../watch/watch_store";
import { WatchDownload } from "../watch/download";
import * as _ from "lodash";

interface ErrorResponse {
  error: {};
}

@Controller("/api/v1/deploy")
export class DeployAPI {
  constructor(
    private readonly clusterStore: ClusterStore,
    private readonly watchStore: WatchStore,
    private readonly downloadService: WatchDownload,
  ) {
  }

  @Get("/desired")
  async getDesiredState(
    @Res() response: Express.Response,
    @HeaderParams("Authorization") auth: string,
  ): Promise<any | ErrorResponse> {
    const span: jaeger.SpanContext = tracer().startSpan("deployAPI.desired");

    const credentials: BasicAuth.Credentials = BasicAuth.parse(auth);

    let cluster;

    try {
      cluster = await this.clusterStore.getFromDeployToken(span.context(), credentials.pass);
    } catch (err) {
      // TODO error type
      response.status(401);
      return {};
    }

    const watches = await this.watchStore.listForCluster(span.context(), cluster.id!);

    const desiredState: string[] = [];

    for (const watch of watches) {
      const params = await this.watchStore.getLatestGeneratedFileS3Params(span, watch.id!);

      const download = await this.downloadService.findDeploymentFile(span.context(), params);
      desiredState.push(download.contents.toString("base64"));
    }

    span.finish();
    response.status(200);
    return {
      present: desiredState,
      missing: [],
    }
  }
}
