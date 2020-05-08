import { BodyParams, Controller, Get, HeaderParams, Put, Req, Res } from "@tsed/common";
import BasicAuth from "basic-auth";
import Express from "express";
import { KotsAppStore, UndeployStatus } from "../../kots_app/kots_app_store";
import { ClusterStore } from "../../cluster";
import { logger } from "../../server/logger";

interface ErrorResponse {
  error: {};
}

@Controller("/api/v1/undeploy")
export class RestoreAPI {
  @Put("/result")
  async putUndeployResult(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
    @HeaderParams("Authorization") auth: string,
    @BodyParams("") body: any,
  ): Promise<any | ErrorResponse> {
    const credentials: BasicAuth.Credentials = BasicAuth.parse(auth);

    let cluster;
    try {
      cluster = await (request.app.locals.stores.clusterStore as ClusterStore).getFromDeployToken(credentials.pass);
    } catch (err) {
      // TODO error type
      response.status(401);
      return {};
    }

    const status = body.is_error ? UndeployStatus.Failed : UndeployStatus.Completed;
    logger.info(`Restore API set RestoreUndeployStatus = ${status} for app ${body.app_id}`);
    const kotsAppStore = request.app.locals.stores.kotsAppStore as KotsAppStore;
    const app = await kotsAppStore.getApp(body.app_id);
    if (app.restoreInProgressName) {
      // Add a delay until we have logic to wait for all pods to be deleted.
      setTimeout(async () => {
        await kotsAppStore.updateAppRestoreUndeployStatus(body.app_id, status);
      }, 20 * 1000);
    }

    return {};
  }
}
