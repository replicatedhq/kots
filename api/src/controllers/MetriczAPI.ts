import * as Express from "express";
import { Controller, Get, Res } from "ts-express-decorators";

interface ErrorResponse {
  error: {};
}

@Controller("/metricz")
export class Metricz {
  @Get("")
  async metricz(@Res() response: Express.Response): Promise<{} | ErrorResponse> {
    response.status(200);

    return {};
  }
}
