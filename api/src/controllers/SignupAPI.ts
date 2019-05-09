import * as Express from "express";
import {
  BodyParams,
  Controller,
  Post,
  Req,
  Res,
} from "ts-express-decorators";

interface SignupRequest {
  email: string;
  firstName: string;
  lastName: string;
  password: string;
}

interface ErrorResponse {
  message: string;
}

@Controller("/api/v1/signup")
export class SignupAPI {
  @Post("")
  public async signup(
    @Res() response: Express.Response,
    @Req() request: Express.Request,
    @BodyParams("") body: any,
  ): Promise<any | ErrorResponse> {
    if (body.email === "" || body.password === "") {
      return {
        message: `Email and password are both required`,
      };
    }

    // const user = await this.userStore.createPasswordUser(body.email, body.password, body.firstName, body.lastName);
    // console.log(user);


    return {};
  }
}
