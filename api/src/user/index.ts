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
import { Context } from "../context";
import { tracer } from "../server/tracing";
import { SessionStore } from "../session/session_store";
import { User } from "./user";
import { GithubNonceStore } from "./store";
import { UserStore } from "./user_store";
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

  // createGitHubAuthToken is the entrypoint from ship-cluster-web when a new github user is signing up
  @Mutation("ship-cloud")
  async createGithubAuthToken(root: any, args: CreateGithubAuthTokenMutationArgs): Promise<AccessToken> {
    const validGithubNonce = await this.validateGithubLoginState(args.state);
    if (!validGithubNonce) {
      throw new ReplicatedError("Invalid GitHub Exchange");
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
      code: args.code,
      redirect_uri: "",
    };

    const accessToken = await oauth2.authorizationCode.getToken(tokenConfig);
    if (accessToken.error) {
      throw new Error(accessToken.error);
    }

    const github = new GitHubApi();
    github.authenticate({
      type: "token",
      token: accessToken.access_token,
    });
    const { data: userData }: { data: GithubUser } = await github.users.get({});

    try {
      const user = await this.getOrCreateGitHubUser(userData.login!, userData.id!, userData.avatar_url!, userData.email!);

      const session = await this.sessionStore.createGithubSession(accessToken.access_token, user.id);

      return {
        access_token: session,
      };
    } catch (e) {
      logger.error(e);
      throw new ReplicatedError("Unable to log in now");
    }
  }

  @Mutation("ship-cloud")
  async refreshGithubTokenMetadata(root: any, args: any, context: Context): Promise<void> {
    const span = tracer().startSpan("mutation.refreshGithubTokenMetadata");
    await this.sessionStore.refreshGithubTokenMetadata(span.context(), context.getGitHubToken(), context.session.id);
    span.finish();
  }

  async getOrCreateGitHubUser(githubUsername: string, githubId: number, githubAvatar: string, email: string): Promise<User> {
    let user = await this.userStore.tryGetGitHubUser(githubId);
    if (user) {
      return user;
    }

    user = await this.userStore.createGitHubUser(githubId, githubUsername.toLowerCase(), githubAvatar, email);
    // const allUsersClusters = await this.clusterStore.listAllUsersClusters(span.context());
    // for (const allUserCluster of allUsersClusters) {
    //   await this.clusterStore.addUserToCluster(span.context(), allUserCluster.id!, shipUser[0].id);
    // }
    return user;
  }

  @Mutation("ship-cloud")
  async createGithubNonce(): Promise<string> {
    const { nonce } = await this.githubNonceStore.createNonce();
    return nonce;
  }

  @Mutation("ship-cloud")
  async trackScmLead(root: any, args: TrackScmLeadMutationArgs, context: Context): Promise<string> {
    const span = tracer().startSpan("mutation.trackScmLead");
    //const { id } = await this.userStore.trackScmLead(span.context(), args.deploymentPreference, args.emailAddress, args.scmProvider);
    const id = "123";
    span.finish();
    return id;
  }

  @Mutation("ship-cloud")
  async logout(root: any, args: any, context: Context): Promise<void> {
    await this.sessionStore.deleteSession(context.session.id);
  }

  async validateGithubLoginState(nonce: string): Promise<boolean> {
    const matchingNonce = await this.githubNonceStore.getNonce(nonce);
    if (!matchingNonce) {
      return false;
    }
    const currentTime = new Date(Date.now()).toUTCString();

    if (isAfter(currentTime, matchingNonce.expire_at!)) {
      return false;
    }

    await this.githubNonceStore.deleteNonce(nonce);

    return true;
  }
}
