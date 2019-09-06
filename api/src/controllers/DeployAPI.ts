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
    let preflight = [];

    for (const app of apps) {
      // app existing in a cluster doesn't always mean deploy.
      // this means that we could possible have a version and/or preflights to run

      // this needs to be updated after the preflight PR is merged
      const pendingPreflightURLs = [];  // todo get this from the store, the fully rendered URLs?
      const deployedAppSequence = app.currentSequence;  // todo this should be 0-n for installed, -1 for no version

      if (pendingPreflightURLs.length > 0) {
        preflight = preflight.concat(pendingPreflightURLs);
      }

      if (deployedAppSequence > -1) {
        const desiredNamespace = ".";
        if (!(desiredNamespace in present)) {
          present[desiredNamespace] = [];
        }

        const rendered = await app.render(''+app.currentSequence, `overlays/downstreams/${cluster.title}`);
        const b = new Buffer(rendered);
        present[desiredNamespace].push(b.toString("base64"));
      }
    }

    response.status(200);
    return {
      present,
      missing,
      preflight,
    }
  }
}
