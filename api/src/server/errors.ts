import * as bugsnag from "bugsnag";
import * as _ from "lodash";
import * as util from "util";
import { logger } from "./logger";

/**
 * ClientErrorDetails is a payload that
 * can be included in a graphql `errors` payload
 * sent to the client.
 */
export interface ClientErrorDetails {
  message: string;
  code: string;
  extra?: {};
}

/**
 * a ReplicatedError's message will be sent down to the client.
 * It is useful for things like Bad Requests and Not Found errors.
 *
 * ReplicatedError is not suitable for
 * 5xx-type server errors, where we don't necessarily want to
 * tell the client what went wrong
 */
export class ReplicatedError extends Error {
  static INTERNAL_ERROR_MESSAGE = "An internal server error has occurred.";

  readonly originalMessage: string;

  constructor(readonly msg: string, readonly code?: string, readonly extra?: {}) {
    super(
      JSON.stringify({
        replicatedMessage: msg || ReplicatedError.INTERNAL_ERROR_MESSAGE,
        replicatedCode: code || msg || "internal_server_error",
        replicatedExtra: extra,
      }),
    );
    this.originalMessage = msg;
    this.extra = extra;
  }

  static forbidden() {
    return new ReplicatedError("Forbidden", "forbidden");
  }

  static isNotFound(err: {}) {
    return err instanceof ReplicatedError && err.isNotFound();
  }

  static notFound() {
    return new ReplicatedError("Not Found", "not_found");
  }

  static requireNonEmpty(item: {}, name?: string) {
    if (_.isEmpty(item)) {
      throw new ReplicatedError(`${name || "value"} may not be empty`, "bad_request", { name });
    }
  }

  static getDetails(error: any): ClientErrorDetails {
    try {
      const parsed = JSON.parse(error.message);
      if (_.has(parsed, "replicatedMessage")) {
        return {
          message: parsed.replicatedMessage,
          code: parsed.replicatedCode,
          extra: parsed.replicatedExtra,
        };
      }
    } catch {
      // ignore
    }

    // hack hack hack, if its a GraphQLError,
    // then it might be a client error in
    // syntax/query
    //
    // I can't wait for this to break
    if (!error.originalError) {
      return {
        message: error.message,
        code: "bad_request",
      };
    }

    // only log it if its an unknown error
    logger.child({ location: "src/server/errors.ts" }).error(util.inspect(error));
    bugsnag.notify(error);

    return {
      message: ReplicatedError.INTERNAL_ERROR_MESSAGE,
      code: "internal_server_error",
    };
  }

  static graphQLUnauthorizedError(): {} {
    return {
      errors: [
        {
          message: "Unauthorized",
          code: "unauthorized",
          locations: [],
        },
      ],
    };
  }
  isNotFound() {
    return this.code === "not_found" || this.originalMessage === ReplicatedError.notFound().originalMessage;
  }
}
