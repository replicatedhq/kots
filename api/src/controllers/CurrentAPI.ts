import Express from "express";
import { Controller, Post, Res, Req, HeaderParams, BodyParams } from "ts-express-decorators";
import BasicAuth from "basic-auth";
import _ from "lodash";

interface ErrorResponse {
  error: {};
}

interface CurrentStateRequest {
  helmApplications: any[],

}
@Controller("/api/v1/current")
export class CurrentAPI {
  @Post("")
  async getDesiredState(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
    @HeaderParams("Authorization") auth: string,
    @BodyParams("") body: CurrentStateRequest,
  ): Promise<{} | ErrorResponse> {
    const credentials: BasicAuth.Credentials = BasicAuth.parse(auth);

    let cluster;

    try {
      cluster = await request.app.locals.stores.clusterStore.getFromDeployToken(credentials.pass);
    } catch (err) {
      // TODO error type
      response.status(401);
      return {};
    }

    for (const helmApplication of body.helmApplications) {
      await request.app.locals.stores.clusterStore.createOrUpdateHelmApplication(cluster.id, helmApplication);
    }
    response.status(200);
    return { }
  }
}
