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

    const watches = await request.app.locals.stores.watchStore.listForCluster(cluster.id!);

    const desiredState: string[] = [];

    for (const watch of watches) {
      const params = await request.app.locals.stores.watchStore.getLatestGeneratedFileS3Params(watch.id!);

      const download = await request.app.locals.stores.watchDownload.findDeploymentFile(params);
      desiredState.push(download.contents.toString("base64"));
    }


    response.status(200);
    return {
      present: desiredState,
      missing: [],
    }
  }
}
