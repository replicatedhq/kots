import GitHubApi from "@octokit/rest";
import { isAfter } from "date-fns";
import simpleOauth from "simple-oauth2";
import { ReplicatedError } from "../../server/errors";
import { logger } from "../../server/logger";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";
import { AdminSignupInfo } from "../";


export type authCode = { code: string };

export function UserMutations(stores: Stores, params: Params) {
  return {
    async loginToAdminConsole(root: any, args: any, context: Context): Promise<AdminSignupInfo> {
      const user = await stores.userStore.tryGetPasswordUser("default-user@none.com");
      if (!user) {
        throw new ReplicatedError("No user was found");
      }
      const validPassword = await user.validatePassword(args.password);
      if (!validPassword) {
        throw new ReplicatedError("Password is incorrect");
      }
      // Passing in "true" as second param to denote this is an admin console user
      const sessionToken = await stores.sessionStore.createPasswordSession(user.id);
      return {
        token: sessionToken,
        userId: user.id,
      };
    },

    async logout(root: any, args: any, context: Context): Promise<void> {
      await stores.sessionStore.deleteSession(context.session.sessionId);
    },
  }
}
