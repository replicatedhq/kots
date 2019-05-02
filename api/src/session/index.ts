import * as jaeger from "jaeger-client";
import * as jwt from "jsonwebtoken";
import { Service } from "ts-express-decorators";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import { Context } from "../server/server";
import { SessionStore } from "../session/store";

@Service()
export class Session {
  constructor(private readonly params: Params, private readonly sessionStore: SessionStore, private readonly tracer: jaeger.Tracer) {}

  // decode attempts to decode the provided token and match it to a valid session.
  // In the case an invalid jwt is passed to it, an empty auth token is returned.
  async decode(token: string): Promise<Context> {
    const span = this.tracer.startSpan("session.decode");
    // not a timing attack since we're not comparing it to the actual value :)
    //tslint:disable-next-line possible-timing-attack
    if (token.length > 0 && token !== "null") {
      try {
        const decoded = jwt.verify(token, this.params.sessionKey);
        const { token: decodedToken, sessionId } = decoded as {
          token: string;
          sessionId: string;
        };

        const session = await this.sessionStore.getGithubSession(span.context(), sessionId);
        if (!session) {
          return this.invalidSession();
        }

        const { user_id: userId, metadata, expiry } = session;

        return {
          auth: decodedToken,
          metadata: JSON.parse(metadata!),
          expiration: expiry,
          userId,
          sessionId,
        };
      } catch (e) {
        // Errors here negligible as they are from jwts not passing verification
        logger.info(e);
      } finally {
        span.finish();
      }
    }

    span.finish();

    return this.invalidSession();
  }

  private invalidSession() {
    return {
      auth: "",
      expiration: new Date(Date.now()),
      userId: "",
      sessionId: "",
    };
  }
}
