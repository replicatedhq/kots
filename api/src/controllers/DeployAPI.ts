import Express from "express";
import { Controller, Get, Res, Req, HeaderParams } from "@tsed/common";
import BasicAuth from "basic-auth";
import _ from "lodash";

interface ErrorResponse {
  error: {};
}

@Controller("/api/v1/deploy")
export class DeployAPI {
  @Get("/desired")
  async getDesiredState(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
    @HeaderParams("Authorization") auth: string,
  ): Promise<any | ErrorResponse> {
    const credentials: BasicAuth.Credentials = BasicAuth.parse(auth);

    let cluster;

    try {
      cluster = await request.app.locals.stores.clusterStore.getFromDeployToken(credentials.pass);
    } catch (err) {
      // TODO error type
      response.status(401);
      return {};
    }

    const apps = await request.app.locals.stores.kotsAppStore.listAppsForCluster(cluster.id);

    const present = {};
    const missing = {};

    for (const app of apps) {
      const desiredNamespace = "test";
      if (!(desiredNamespace in present)) {
        present[desiredNamespace] = [];
      }

      const rendered = await app.render(''+app.currentSequence, `overlays/downstreams/${cluster.title}`);
      const b = new Buffer(rendered);
      present[desiredNamespace].push(b.toString("base64"));
    }

    response.status(200);
    return {
      present,
      missing,
    }
  }
}
