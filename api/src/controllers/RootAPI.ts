import * as Express from "express";
import { Controller, Get, Res } from "ts-express-decorators";

@Controller("/")
export class RootAPI {
  @Get("")
  async root(@Res() response: Express.Response): Promise<any> {
    response.status(200);

    return { };
  }
}
