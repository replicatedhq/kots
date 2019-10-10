import Express from "express";
import { Controller, Put, Get, Res, Req, HeaderParams, BodyParams } from "@tsed/common";
import BasicAuth from "basic-auth";
import _ from "lodash";

interface ErrorResponse {
  error: {};
}

@Controller("/api/v1/deploy")
export class DeployAPI {
  @Put("/result")
  async putDeployResult(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
    @HeaderParams("Authorization") auth: string,
    @BodyParams("") body: any,
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

    const output = {
      dryRun: {
        stderr: body.dryrun_stderr,
        stdout: body.dryrun_stdout,
      },
      apply: {
        stderr: body.apply_stderr,
        stdout: body.apply_stdout,
      },
    };

    const apps = await request.app.locals.stores.kotsAppStore.listAppsForCluster(cluster.id);

    // sequence really should be passed down to operator and returned from it
    const downstreamVersion = await request.app.locals.stores.kotsAppStore.getCurrentVersion(body.app_id, cluster.id);

    await request.app.locals.stores.kotsAppStore.updateDownstreamDeployStatus(body.app_id, cluster.id, downstreamVersion.sequence, body.is_error, output);

    return {};
  }

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

    const present: any[] = [];
    const missing = {};
    let preflight = [];

    for (const app of apps) {
      // app existing in a cluster doesn't always mean deploy.
      // this means that we could possible have a version and/or preflights to run

      // this needs to be updated after the preflight PR is merged
      const pendingPreflightURLs = await request.app.locals.stores.preflightStore.getPendingPreflightUrls();
      const deployedKotsAppVersion = await request.app.locals.stores.kotsAppStore.getCurrentVersion(app.id, cluster.id);
      const deployedAppSequence = deployedKotsAppVersion && deployedKotsAppVersion.sequence;
      if (pendingPreflightURLs.length > 0) {
        preflight = preflight.concat(pendingPreflightURLs);
      }

      if (deployedAppSequence > -1) {
        const desiredNamespace = ".";

        const rendered = await app.render(''+app.currentSequence, `overlays/downstreams/${cluster.title}`);
        const b = new Buffer(rendered);

        const applicationManifests = {
          "app_id": app.id,
          namespace: desiredNamespace,
          manifests: b.toString("base64"),
        }

        present.push(applicationManifests);
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
