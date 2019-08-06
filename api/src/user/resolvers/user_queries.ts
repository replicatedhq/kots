import { Stores } from "../../schema/stores";
import { Context } from "../../context";
import GitHubApi from "@octokit/rest";
import { ReplicatedError } from "../../server/errors";

export function UserQueries(stores: Stores) {
  return {
    async userInfo(root: any, args: any, context: Context) {
      if (context.sessionType() === "github") {
        const github = new GitHubApi();
        github.authenticate({
          type: "token",
          token: context.getGitHubToken(),
        });

        const githubUser = await github.users.get({});
        return {
          avatarUrl: githubUser.data.avatar_url,
          username: githubUser.data.login
        }
      } else if (context.sessionType() === "ship") {
        return {
          avatarUrl: "",
          username: "test"
        }
      } else {
        throw new ReplicatedError(`Unknown session type: ${context.sessionType()}`);
      }
    },

    async isSecured(root: any, args: any, context: Context) {
      return stores.userStore.checkSecuredStatus();
    }
  }
}


