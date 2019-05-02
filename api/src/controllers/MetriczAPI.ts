import * as Express from "express";
import { instrumented } from "monkit";
import { Controller, Get, Res } from "ts-express-decorators";

interface ErrorResponse {
  error: {};
}

@Controller("/metricz")
export class Metricz {
  @Get("")
  @instrumented()
  async metricz(@Res() response: Express.Response): Promise<{} | ErrorResponse> {
    response.status(200);

    return {};
  }
}
