import { Err, GlobalErrorHandlerMiddleware, OverrideProvider, Req, Res } from "@tsed/common";

@OverrideProvider(GlobalErrorHandlerMiddleware)
export class ErrorHandler extends GlobalErrorHandlerMiddleware {

  use(@Err() error: any,
    @Req() request: Req,
    @Res() response: Res): any {

    console.log("ERROR!");
    console.log(error);

    return super.use(error, request, response);
  }
}