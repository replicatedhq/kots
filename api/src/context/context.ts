import { SessionStore, Session } from "../session";
import { getPostgresPool } from "../util/persistence/db";
import { Params } from "../server/params";
import { ReplicatedError } from "../server/errors";
import { isAfter } from "date-fns";

export class Context {
  public session: Session;

  public static async fetch(token: string): Promise<Context> {
    const pool = await getPostgresPool();
    const params = await Params.getParams();
    const sessionStore = new SessionStore(pool, params);

    const context = new Context();
    context.session = await sessionStore.decode(token);

    return context;
  }

  public getGitHubToken(): string {
    return this.session.scmToken;
  }

  public hasValidSession(): ReplicatedError | null {
    if (this.getGitHubToken().length === 0) {
      return new ReplicatedError("Unauthorized", "401");
    }

    const currentTime = new Date(Date.now()).toUTCString();
    if (isAfter(currentTime, this.session.expiresAt)) {
      return new ReplicatedError("Expired session", "401");
    }

    return null
  }
}
