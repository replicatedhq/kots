import { Controller, Put, Post, BodyParams } from "@tsed/common";
import { MultipartFile } from "@tsed/multipartfiles";

interface CreateAppBody {
  name: string
}

interface UpdateAppBody {
  slug: string;
}

@Controller("/api/v1/kots")
export class KotsAPI {
  @Post("/")
  async kotsUploadCreate(
    @MultipartFile("file") file: Express.Multer.File,
    @BodyParams("") body: CreateAppBody,
  ): Promise<any> {

    // body.name is the name of the app

    // file.filename is a locally-stored (need to read it) copy of the archive

    // this should create the application in pg
    // upload the version to s3
    // create a version in pg

    // return the url for the app

    return {
      uri: "https://www.google.com",
    };
  }

  @Put("/")
  async kotsUploadUpdate(
    @MultipartFile("file") file: Express.Multer.File,
    @BodyParams("") body: UpdateAppBody,
  ): Promise<any> {

    // body.slug is the slug of the app to update

    // file.filename is a locally-stored (need to read it) copy of the archive

    // this should create the application in pg
    // upload the version to s3
    // create a version in pg

    // return the url for the app

    return {
      uri: "https://www.google.com",
    };
  }
}
