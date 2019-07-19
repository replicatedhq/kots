import Express from "express";
import { Controller, Get, PathParams, Res, Req } from "ts-express-decorators";

@Controller("/api/v1/download")
export class WatchDownloadAPI {
  @Get("/:watchId")
  async downloadDeploymentYAML(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
    @PathParams("watchId") watchId: string
  ): Promise<void> {
    const watch = await response.locals.context.getWatch(watchId);

    const { filename, contents, contentType } = await request.app.locals.stores.watchDownload.downloadDeploymentYAML(watch);

    response.setHeader("Content-Disposition", `attachment; filename=${filename}`);
    response.setHeader("Content-Type", contentType);
    response.send(contents);
  }

  @Get("/:watchId/:sequence")
  async downloadDeploymentYAMLForSequence(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
    @PathParams("watchId") watchId: string,
    @PathParams("sequence") sequence: number):
  Promise<void> {
    const watch = await response.locals.context.getWatch(watchId);

    const { filename, contents, contentType } = await request.app.locals.stores.watchDownload.downloadDeploymentYAMLForSequence(watch, sequence);

    response.setHeader("Content-Disposition", `attachment; filename=${filename}`);
    response.setHeader("Content-Type", contentType);
    response.send(contents);
  }
}
