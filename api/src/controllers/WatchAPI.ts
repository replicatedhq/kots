import Express from "express";
import { Controller, Get, Res, Req, QueryParams, PathParams } from "ts-express-decorators";

interface ErrorResponse {
  error: {};
}

@Controller("/api/v1/watch")
export class WatchAPI {
  @Get("/:upstreamId/upstream.yaml")
  async getUpstreamWatch(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
    @PathParams("upstreamId") upstreamId: string,
    @QueryParams("token") token: string,
  ): Promise<any | ErrorResponse> {

    const watch = await request.app.locals.stores.watchStore.findUpstreamWatch(token, upstreamId);

    const params = await request.app.locals.stores.watchStore.getLatestGeneratedFileS3Params(watch.id!);

    const download = await request.app.locals.stores.watchDownload.findDeploymentFile(params);

    response.status(200);

    return download.contents.toString("utf-8");
  }
}
