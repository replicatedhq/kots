import { Controller, Put } from "@tsed/common";
import { MultipartFile } from "@tsed/multipartfiles";

@Controller("/api/v1/kots")
export class KotsAPI {
  @Put("/")
  async kotsPut(
    @MultipartFile("kotsapp.tar.gz") file: Express.Multer.File
  ): Promise<any> {
    console.log("Luke i am father")
    console.log(typeof file)
    return {};
  }
}
