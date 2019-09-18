import Express from "express";
import {
  BodyParams,
  Controller,
  Post,
  Req,
  Res,
} from "@tsed/common";

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

    const user = await request.app.locals.stores.userStore.createPasswordUser(body.email, body.password, body.firstName, body.lastName);

    const sessionToken = await request.app.locals.stores.sessionStore.createPasswordSession(user.id);

    response.status(201);
    return {
      token: sessionToken,
    };
  }
}
