import { Controller, Get, Req, Res } from "@tsed/common";
import Express from "express";
import { DatabaseInfo } from "../healthz/healthz_store";

interface HealthzResponse {
  status?: DatabaseInfo
  version?: string
  gitSha?: string
}

@Controller("/healthz")
export class HealthzAPI {

  @Get("")
  public async getDatabaseInfo(
    @Req() request: Express.Request,
    @Res() response: Express.Response,
  ): Promise<any> {
    let res: HealthzResponse = {gitSha: process.env.COMMIT}

    res.status = await request.app.locals.stores.healthzStore.getHealthz();
    if (!res.status!.database.connected || !res.status!.storage.available) {
      response.status(419);
      return res;
    }

    return res;
  }
}
