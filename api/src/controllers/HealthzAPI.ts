import { Controller, Get, Req } from "@tsed/common";
import Express from "express";

@Controller("/healthz")
export class HealthzAPI {

  @Get("")
  public async getDatabaseInfo(
    @Req() request: Express.Request
  ): Promise<any> {
    return request.app.locals.stores.healthzStore.getHealthz();
  }
}
