import * as Express from "express";
import { Controller, Get, PathParams, Res, Req } from "ts-express-decorators";

@Controller("/api/install")
export class InstallAPI {
  @Get("/:id/:token")
  async renderedOperatorInstall(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
    @PathParams("id") clusterId: string,
    @PathParams("token") token: string,
  ): Promise<void | string> {
    const cluster = await request.app.locals.stores.clusterStore.getCluster(clusterId);
    if (cluster.shipOpsRef!.token !== token) {
      response.status(404);
      return;
    }

    const manifests = await request.app.locals.stores.clusterStore.getShipInstallationManifests(cluster.id!);

    response.setHeader("Content-Type", "text/plain");
    response.send(`${manifests}`);

    response.status(200);
  }
}
