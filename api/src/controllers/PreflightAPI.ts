import { Request, Response } from "express";
import { Controller, Get, PathParams, Put, Res, Req} from "ts-express-decorators";
import jsYaml from "js-yaml";

@Controller("/api/v1/preflight")
export class PreflightAPI {
  @Get("/:watchId/:downstream_cluster")
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