import { Err, GlobalErrorHandlerMiddleware, OverrideProvider, Req, Res } from "@tsed/common";
import { getBugsnagClient } from "../server/bugsnagClient";

@OverrideProvider(GlobalErrorHandlerMiddleware)
export class ErrorHandler extends GlobalErrorHandlerMiddleware {

  use(
    @Err() error: any,
    @Req() request: Req,
    @Res() response: Res
  ): any {

    const bugsnagClient = getBugsnagClient();
    if (bugsnagClient) {
      bugsnagClient.notify(error, {
        request,
        severity: "error",
      });
    }
    console.error(error);
    super.use(error, request, response);
  }
}
