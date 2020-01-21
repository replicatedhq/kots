import { BodyParams, Controller, Get, HeaderParams, Put, Req, Res } from "@tsed/common";
import BasicAuth from "basic-auth";
import Express from "express";
import _ from "lodash";

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
      cluster = await request.app.locals.stores.clusterStore.getFromDeployToken(credentials.pass);
    } catch (err) {
      // TODO error type
      response.status(401);
      return {};
    }

    console.log("+++ UNDEPLOY", JSON.stringify(body, null, 2));

    return {};
  }
}
