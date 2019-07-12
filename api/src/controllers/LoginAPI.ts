import * as Express from "express";
import {
  BodyParams,
  Controller,
  Post,
  Req,
  Res,
} from "ts-express-decorators";

interface LoginRequest {
  email: string;
  password: string;
}

interface ErrorResponse {
  message: string;
}

@Controller("/api/v1/login")
export class LoginAPI {
  @Post("")
  public async login(
    @Res() response: Express.Response,
    @Req() request: Express.Request,
    @BodyParams("") body: any,
  ): Promise<any | ErrorResponse> {
    if (body.email === "" || body.password === "") {
      return {
        message: `Email and password are both required`,
      };
    }

    const user = await request.app.locals.stores.userStore.tryGetPasswordUser(body.email);
    if (!user) {
      response.status(401);
      return {};
    }

    if (!await user.validatePassword(body.password)) {
      response.status(401);
      return {};
    }

    const sessionToken = await request.app.locals.stores.sessionStore.createPasswordSession(user.id);

    response.status(200);
    return {
      token: sessionToken,
    };
  }
}
