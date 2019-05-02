import * as GitHubApi from "@octokit/rest";
import { isAfter } from "date-fns";
import * as jaeger from "jaeger-client";
import * as simpleOauth from "simple-oauth2";
import { Service } from "ts-express-decorators";
import { AccessToken, CreateGithubAuthTokenMutationArgs, GithubUser, TrackScmLeadMutationArgs } from "../generated/types";
import { Mutation } from "../schema/decorators";
import { ReplicatedError } from "../server/errors";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import { Context } from "../server/server";
import { tracer } from "../server/tracing";
import { SessionStore } from "../session/store";
import { storeTransaction } from "../util/persistence/db";
import { authorized } from "./decorators";
import { UserModel } from "./models";
import { GithubNonceStore, UserStore } from "./store";
import { ClusterStore } from "../cluster/cluster_store";

export type authCode = { code: string };
@Service()
export class Auth {
  constructor(
    private readonly params: Params,
    private readonly userStore: UserStore,
    private readonly githubNonceStore: GithubNonceStore,
    private readonly clusterStore: ClusterStore,
    private readonly sessionStore: SessionStore,
  ) {}

  @Mutation("ship-cloud")
  async createGithubAuthToken(root: any, { state, code }: CreateGithubAuthTokenMutationArgs): Promise<AccessToken> {
    const span = tracer().startSpan("mutation.createGithubAuthToken");

    const validGithubNonce = await this.validateGithubLoginState(span.context(), state);
    if (!validGithubNonce) {
      throw new ReplicatedError("Ship Cloud Access Denied");
    }

    const githubClientId = this.params.githubClientId;
    const githubClientSecret = this.params.githubClientSecret;

    const oauth2 = simpleOauth.create({
      client: {
        id: githubClientId,
        secret: githubClientSecret,
      },
      auth: {
        tokenHost: "https://github.com",
        tokenPath: "/login/oauth/access_token",
        authorizePath: "/login/oauth/authorize",
      },
    });

    const tokenConfig = {
      code,
      redirect_uri: "",
    };

    const accessToken = await oauth2.authorizationCode.getToken(tokenConfig);

    if (accessToken.error) {
      span.finish();
      throw new Error(accessToken.error);
    }

    const github = new GitHubApi();
    github.authenticate({
      type: "token",
      token: accessToken.access_token,
    });
    const { data: userData }: { data: GithubUser } = await github.users.get({});

    try {
      const [user] = await this.maybeCreateUser(span.context(), userData.login!, userData.id!, userData.avatar_url!, userData.email!);

      const session = await this.sessionStore.createGithubSession(span.context(), accessToken.access_token, user.id);

      span.finish();

      return {
        access_token: session,
      };
    } catch (e) {
      logger.error(e);
      throw new ReplicatedError("Ship Cloud Access Denied");
    }
  }

  @Mutation("ship-cloud")
  @authorized()
  async refreshGithubTokenMetadata(root: any, args: any, context: Context): Promise<void> {
    const span = tracer().startSpan("mutation.refreshGithubTokenMetadata");
    await this.sessionStore.refreshGithubTokenMetadata(span.context(), context.auth, context.sessionId);
    span.finish();
  }

  async maybeCreateUser(ctx: jaeger.SpanContext, githubUsername: string, githubId: number, githubAvatar: string, email?: string): Promise<UserModel[]> {
    const span: jaeger.Span = tracer().startSpan("maybeCreateUser", { childOf: ctx });

    span.setTag("githubUsername", githubUsername);
    span.setTag("githubId", githubId);

    const lowerUsername = githubUsername.toLowerCase();

    const user = await this.userStore.getUser(span.context(), githubId);

    if (!user.length) {
      await storeTransaction(this.userStore, async store => {
        await store.createGithubUser(span.context(), githubId, lowerUsername, githubAvatar, email);
        await store.createShipUser(span.context(), githubId, lowerUsername);

        const shipUser = await store.getUser(span.context(), githubId);
        const allUsersClusters = await this.clusterStore.listAllUsersClusters(span.context());
        for (const allUserCluster of allUsersClusters) {
          await this.clusterStore.addUserToCluster(span.context(), allUserCluster.id!, shipUser[0].id);
        }
      });

      return this.userStore.getUser(span.context(), githubId);
    }

    if (email) {
      await this.userStore.updateGithubUserEmail(span.context(), githubId, email);
    }

    span.finish();

    return user;
  }

  @Mutation("ship-cloud")
  async createGithubNonce(): Promise<string> {
    const span = tracer().startSpan("mutation.createCode");

    const { nonce } = await this.githubNonceStore.createNonce(span.context());

    span.finish();

    return nonce;
  }

  @Mutation("ship-cloud")
  async trackScmLead(root: any, args: TrackScmLeadMutationArgs, context: Context): Promise<string> {
    const span = tracer().startSpan("mutation.trackScmLead");
    const { id } = await this.userStore.trackScmLead(span.context(), args.deploymentPreference, args.emailAddress, args.scmProvider);
    span.finish();
    return id;
  }

  @Mutation("ship-cloud")
  @authorized()
  async logout(root: any, args: any, { sessionId }: Context): Promise<null> {
    const span = tracer().startSpan("mutation.logout");

    await this.sessionStore.deleteSession(span.context(), sessionId);

    return null;
  }

  async validateGithubLoginState(ctx: jaeger.SpanContext, nonce: string): Promise<boolean> {
    const span: jaeger.Span = tracer().startSpan("validateGithubLoginState", { childOf: ctx });

    const matchingNonce = await this.githubNonceStore.getNonce(span.context(), nonce);
    if (!matchingNonce) {
      return false;
    }
    const currentTime = new Date(Date.now()).toUTCString();

    if (isAfter(currentTime, matchingNonce.expire_at!)) {
      return false;
    }

    await this.githubNonceStore.deleteNonce(span.context(), nonce);

    return true;
  }
}
