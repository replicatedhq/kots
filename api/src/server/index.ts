import { Server } from "./server";
import {ServerLoader, ServerSettings, OverrideMiddleware, LogIncomingRequestMiddleware, Res, Req} from "@tsed/common";
import uuid from "uuid";

import Express from "express";
import { logger, TSEDVerboseLogging } from "./logger";

@OverrideMiddleware(LogIncomingRequestMiddleware)
export class CustomLogIncomingRequestMiddleware extends LogIncomingRequestMiddleware {
  use(@Req() request: Express.Request) {
    request.id = uuid.v4();

    if (request.path === "/healthz") {
      this.configureRequest(request, true);

      return;
    }

    super.use(request);
  }

  // mostly copy pasted, added suppress flag for quieter healthz logging
  protected configureRequest(request: Express.Request, suppress?: boolean) {
    const verbose = (req: Express.Request) => this.requestToObject(req);
    const info = (req: Express.Request) => this.minimalRequestPicker(req);

    if (!suppress) {
      return;
    }

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
        logger.error({ msg: "error logging message", error: err });
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
      };
    }
  }
}

// tslint:disable no-console
function main() {
  new Server()
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
