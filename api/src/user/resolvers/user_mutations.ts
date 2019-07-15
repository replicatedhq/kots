import * as GitHubApi from "@octokit/rest";
import { isAfter } from "date-fns";
import * as simpleOauth from "simple-oauth2";
import { ReplicatedError } from "../../server/errors";
import { logger } from "../../server/logger";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";
import { GitHubUser, AccessToken } from "../";

export type authCode = { code: string };

export function UserMutations(stores: Stores, params: Params) {
  return {
    async createGithubAuthToken(root: any, args: any): Promise<AccessToken> {
      const matchingNonce = await stores.githubNonceStore.getNonce(args.state);
      if (!matchingNonce) {
        throw new ReplicatedError("Invalid GitHub Exchange, no matching nonce found.");
      }
      const currentTime = new Date(Date.now()).toUTCString();

      if (isAfter(currentTime, matchingNonce.expire_at!)) {
        throw new ReplicatedError("Invalid GitHub Exchange, nonce has expired.");
      }

      await stores.githubNonceStore.deleteNonce(args.state);

      const githubClientId = params.githubClientId;
      const githubClientSecret = params.githubClientSecret;

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
      const response = await github.users.get({});

      try {
        let user = await stores.userStore.tryGetGitHubUser(response.data.id);
        if (!user) {
          user = await stores.userStore.createGitHubUser(response.data.id, response.data.login.toLowerCase(), response.data.avatar_url, response.data.email);
          // const allUsersClusters = await this.clusterStore.listAllUsersClusters(span.context());
          // for (const allUserCluster of allUsersClusters) {
          //   await this.clusterStore.addUserToCluster(span.context(), allUserCluster.id!, shipUser[0].id);
          // }
        }
        await stores.userStore.updateLastLogin(user.id);
        const session = await stores.sessionStore.createGithubSession(user.id, github, accessToken.access_token);

        return {
          access_token: session,
        };
      } catch (err) {
        logger.error({ msg: err.message, err, token: accessToken.access_token });
        throw new ReplicatedError("Unable to log in now");
      }
    },

    async createGithubNonce(): Promise<string> {
      const nonce = await stores.githubNonceStore.createNonce();
      return nonce.nonce;
    },

    async trackScmLead(root: any, args: any, context: Context): Promise<string> {
      return await stores.userStore.trackScmLead(args.deploymentPreference, args.emailAddress, args.scmProvider);
    },

    async logout(root: any, args: any, context: Context): Promise<void> {
      await stores.sessionStore.deleteSession(context.session.sessionId);
    },
  }
}
