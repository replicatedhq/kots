import Express from "express";
import { BodyParams, Controller, Req, Res, Put } from "ts-express-decorators";
import { MultipartFile, MulterOptions } from "@tsed/multipartfiles";

@Controller("api/v1/kots")
export class KotsAPI {
  @Put("")
  async kotsPut(
    @Res() response: Express.Response,
    @Req() request: Express.Request,
    @MultipartFile("file") file: any,
  ): Promise<any> {
    console.log(file)
    return {};
  }
}
