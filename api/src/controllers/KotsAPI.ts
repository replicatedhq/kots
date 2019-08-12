import { Controller, Put } from "@tsed/common";
import { MultipartFile } from "@tsed/multipartfiles";

@Controller("api/v1/kots")
export class KotsAPI {
  @Put("")
  async kotsPut(
    @MultipartFile("file") file: any,
  ): Promise<any> {
    console.log(file)
    return {};
  }
}
