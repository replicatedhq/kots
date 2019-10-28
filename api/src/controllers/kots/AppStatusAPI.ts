import Express from "express";
import { Controller, Put, Get, Res, Req, HeaderParams, BodyParams } from "@tsed/common";
import { Cluster } from "../../cluster";
import BasicAuth from "basic-auth";
import _ from "lodash";
import { KotsAppStatusStore } from "../../kots_app/kots_app_status_store";

interface ErrorResponse {
  error: {};
}

@Controller("/api/v1/appstatus")
export class AppStatusAPI {
  @Put("/")
  async putAppStatus(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
    @HeaderParams("Authorization") auth: string,
    @BodyParams("") body: any,
  ): Promise<void | ErrorResponse> {
    const credentials: BasicAuth.Credentials = BasicAuth.parse(auth);

    try {
      await request.app.locals.stores.clusterStore.getFromDeployToken(credentials.pass);
    } catch (err) {
      // TODO error type
      response.status(401);
      return;
    }

    const kotsAppStatusStore: KotsAppStatusStore = request.app.locals.stores.kotsAppStatusStore;
    await kotsAppStatusStore.setKotsAppStatus(body.app_id, body.resource_states, body.updated_at);
    response.status(201);
  }
}
