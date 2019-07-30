import { Request, Response } from "express";
import { Controller, Get, PathParams, Put, Post, Res, Req} from "ts-express-decorators";

@Controller("/preflight")
export class PreflightAPI {
  @Get("/:watchId/:preflight")
  async getPreflightStatus(
    @Req() request: Request,
    @Res() response: Response,
    @PathParams("watchId") watchId: string,
    @PathParams("preflight") preflight: string
  ): Promise<void> {

  }

  @Put("/:watchId/:preflight")
  async putPreflightStatus(
    @Req() request: Request,
    @Res() response: Response,
    @PathParams("watchId") watchId: string,
    @PathParams("preflight") preflight: string
  ): Promise<void> {

  }

  @Post("/:watchId/:preflight")
  async postPreflightStatus(
    @Req() request: Request,
    @Res() response: Response,
    @PathParams("watchId") watchId: string,
    @PathParams("preflight") preflight: string
  ): Promise<void> {

  }

}