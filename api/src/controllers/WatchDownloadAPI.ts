import * as Express from "express";
import { Controller, Get, PathParams, Res, Req } from "ts-express-decorators";

@Controller("/api/v1/download")
export class WatchDownloadAPI {
  @Get("/:watchId")
  async downloadDeploymentYAML(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
    @PathParams("watchId") watchId: string
  ): Promise<void> {
    const { userId = "" } = response.locals.context || {};
    const watch = await request.app.locals.stores.watchStore.findUserWatch(userId, { id: watchId });

    const { filename, contents, contentType } = await request.app.locals.stores.watchDownload.downloadDeploymentYAML(watch.id!);

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
    const { userId = "" } = response.locals.context || {};
    const watch = await request.app.locals.stores.watchStore.findUserWatch(userId, { id: watchId });

    const { filename, contents, contentType } = await request.app.locals.stores.watchDownload.downloadDeploymentYAMLForSequence(watch.id!, sequence);

    response.setHeader("Content-Disposition", `attachment; filename=${filename}`);
    response.setHeader("Content-Type", contentType);
    response.send(contents);
  }
}
