import { Controller, Put, BodyParams } from "@tsed/common";
import { MultipartFile } from "@tsed/multipartfiles";

@Controller("/api/v1/kots")
export class KotsAPI {
  @Put("/")
  async kotsPut(
    @MultipartFile("file") file: Express.Multer.File,
    @BodyParams("") body: any,
  ): Promise<any> {
    console.log("Luke i am father")
    console.log(file);
    console.log(body);
    return {};
  }
}
