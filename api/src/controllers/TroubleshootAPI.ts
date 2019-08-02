import Express from "express";
import { Controller, Get, Res, Req, PathParams } from "ts-express-decorators";
import { TroubleshootStore } from "../troubleshoot";

interface ErrorResponse {
  error: {};
}

@Controller("/api/v1/troubleshoot")
export class TroubleshootAPI {
  @Get("/:partA/:partB")
  async getSpec(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
    @PathParams("partA") partA: string,
    @PathParams("partB") partB: string,
  ): Promise<any | ErrorResponse> {

    const slug = partA + "/" + partB;
    const collector = await request.app.locals.stores.troubleshootStore.getCollectorForWatchSlug(slug);

    response.status(200);
    return collector.spec;
  }
}
