import { Controller, Get, Req, Res } from "@tsed/common";
import Express from "express";

@Controller("/healthz")
export class HealthzAPI {

  @Get("")
  public async getDatabaseInfo(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
  ): Promise<any> {
    const res = await request.app.locals.stores.healthzStore.getHealthz();
    if (!res.database.connected || !res.storage.available) {
      response.status(419);
      return {};
    }

    return res;
  }
}
