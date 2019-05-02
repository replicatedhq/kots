import * as Express from "express";
import { instrumented } from "monkit";
import { Controller, Get, PathParams, Res } from "ts-express-decorators";
import * as jaeger from "jaeger-client";
import { tracer } from "../server/tracing";
import { ClusterStore } from "../cluster/cluster_store";

@Controller("/api/install")
export class InstallAPI {
  constructor(
    private readonly clusterStore: ClusterStore,
  ) {
  }

  @Get("/:id/:token")
  @instrumented()
  async renderedOperatorInstall(
    @Res() response: Express.Response,
    @PathParams("id") clusterId: string,
    @PathParams("token") token: string,
  ): Promise<void | string> {
    const span: jaeger.SpanContext = tracer().startSpan("installAPI.renderedOperatorInstall");

    const cluster = await this.clusterStore.getCluster(span.context(), clusterId);
    if (cluster.shipOpsRef!.token !== token) {
      response.status(404);
      return;
    }

    const manifests = await this.clusterStore.getShipInstallationManifests(span.context(), cluster.id!);

    response.setHeader("Content-Type", "text/plain");
    response.send(`${manifests}`);
    span.finish();

    response.status(200);
  }
}
