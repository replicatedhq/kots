import _ from "lodash";

/**
 * ClientErrorDetails is a payload that
 * can be included in a graphql `errors` payload
 * sent to the client.
 */
export interface ClientErrorDetails {
  msg: string;
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

  constructor(readonly msg: string) {
    super(
      JSON.stringify({
        replicatedMessage: msg || ReplicatedError.INTERNAL_ERROR_MESSAGE,
      }),
    );
    this.originalMessage = msg;
  }

  static notFound() {
    return new ReplicatedError("not found");
  }

  static forbidden() {
    return new ReplicatedError("Forbidden");
  }

  static requireNonEmpty(item: {}, name?: string) {
    if (_.isEmpty(item)) {
      throw new ReplicatedError(`${name || "value"} may not be empty`);
    }
  }

  static getDetails(error: any): ClientErrorDetails {
    try {
      const parsed = JSON.parse(error.message);
      if (_.has(parsed, "replicatedMessage")) {
        return {
          msg: parsed.replicatedMessage,
        };
      }
    } catch {
      // ignore
    }

    // hack hack hack, if its a GraphQLError,
    // then it might be a client error in
    // syntax/query
    if (!error.originalError) {
      return {
        msg: error.message,
      };
    }

    return {
      msg: ReplicatedError.INTERNAL_ERROR_MESSAGE,
    };
  }

  static graphQLUnauthorizedError(): {} {
    return {
      errors: [
        {
          message: "Unauthorized",
          locations: [],
        },
      ],
    };
  }

  static isNotFound(err: any): boolean {
    if (err instanceof ReplicatedError && err.originalMessage === "not found") {
      return true;
    }
    return false;
  }
}
