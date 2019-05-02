import { OverrideMiddleware, Req, Res } from "ts-express-decorators";

import { LogIncomingRequestMiddleware } from "ts-express-decorators/lib/mvc/components/LogIncomingRequestMiddleware";
import * as uuid from "uuid";

import * as Express from "express";
import { logger, TSEDVerboseLogging } from "./logger";
import * as vendor from "./server";
import * as tracing from "./tracing";

@OverrideMiddleware(LogIncomingRequestMiddleware)
export class CustomLogIncomingRequestMiddleware extends LogIncomingRequestMiddleware {
  use(@Req() request: Express.Request, @Res() response: Express.Response) {
    request.id = uuid.v4();

    if (request.path === "/healthz") {
      this.configureRequest(request, true);

      return;
    }

    super.use(request, response);
  }

  // mostly copy pasted, added suppress flag for quieter healthz logging
  protected configureRequest(request: Express.Request, suppress?: boolean) {
    request.tsedReqStart = new Date();

    const verbose = (req: Express.Request) => this.requestToObject(req);
    const info = (req: Express.Request) => this.minimalRequestPicker(req);

    // tslint:disable no-void-expression
    request.log = {
      info: (obj: any) => logger.debug(this.stringify(request, info)(obj)),
      debug: (obj: any) => logger.debug(this.stringify(request, verbose)(obj)),
      warn: (obj: any) => logger.warn(this.stringify(request, verbose)(obj)),
      error: (obj: any) => logger.error(this.stringify(request, verbose)(obj)),
      trace: (obj: any) => logger.debug(this.stringify(request, verbose)(obj)),
    };

    if (!suppress) {
      return;
    }

    request.log = {
      info: (obj: any) => logger.debug(this.stringify(request, info)(obj)),
      debug: (obj: any) => logger.debug(this.stringify(request, verbose)(obj)),
      warn: (obj: any) => logger.debug(this.stringify(request, verbose)(obj)),
      error: (obj: any) => logger.error(this.stringify(request, verbose)(obj)),
      trace: (obj: any) => logger.debug(this.stringify(request, verbose)(obj)),
    };
    // tslint:enable no-void-expression
  }

  // pretty much copy-pasted, but hooked into TSEDVerboseLogging from above to control multiline logging
  protected stringify(request: Express.Request, propertySelector: (e: Express.Request) => {}): (scope: {}) => string {
    return inscope => {
      let scope = inscope;
      if (!scope) {
        scope = {};
      }

      if (typeof scope === "string") {
        scope = { message: scope };
      }

      scope = { ...scope, ...propertySelector(request) };
      try {
        if (TSEDVerboseLogging) {
          return JSON.stringify(scope, null, 2);
        }

        return JSON.stringify(scope);
      } catch (err) {
        logger.error({ error: err });
      }

      return "";
    };
  }

  protected requestToObject(request: Express.Request) {
    if (request.originalUrl === "/healthz" || request.url === "/healthz") {
      return {
        url: "/healthz",
      };
    }

    if (TSEDVerboseLogging) {
      return {
        reqId: request.id,
        method: request.method,
        url: request.originalUrl || request.url,
        duration: new Date().getTime() - request.tsedReqStart.getTime(),
        headers: request.headers,
        body: request.body,
        query: request.query,
        params: request.params,
      };
    } else {
      return {
        reqId: request.id,
        method: request.method,
        url: request.originalUrl || request.url,
        duration: new Date().getTime() - request.tsedReqStart.getTime(),
      };
    }
  }
}

// tslint:disable no-console
function main() {
  tracing.bootstrap();

  // todo use a library for argv
  // pass "--check" to spin up the server, ensure injection works, then shut down.
  // catches issues in packaged binary like
  // https://replicated.slack.com/archives/C9QQD9LHK/p1536791961000100
  if (process.argv.indexOf("--check") !== -1) {
    const timeoutMs = 1000;
    console.log(`--check flag was passed, will exit in ${timeoutMs}ms`);
    setTimeout(() => {
      console.log("exiting (0) because check flag was passed");
      process.exit(0);
    }, timeoutMs);
  }

  new vendor.Server()
    .start()
    .then(() => {
      process.on("SIGINT", () => {
        console.log("received interrupt...");
        setTimeout(() => {
          process.exit(137);
        }, 100);
      });
    })
    .catch(err => {
      console.log("received error...", err);
      setTimeout(() => {
        process.exit(137);
      }, 100);
    });
}

main();
