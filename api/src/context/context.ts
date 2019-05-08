import { SessionStore, Session } from "../session";
import { PostgresWrapper, getPostgresPool } from "../util/persistence/db";
import { Params } from "../server/params";

export class Context {
  public session: Session;

  public static async fetch(token: string): Promise<Context> {
    const wrapper = new PostgresWrapper(await getPostgresPool());
    const params = await Params.getParams();
    const sessionStore = new SessionStore(wrapper, params);

    const context = new Context();
    context.session = await sessionStore.decode(token);

    return context;
  }

  public getGitHubToken(): string {
    return this.session.scmToken;
  }
}
