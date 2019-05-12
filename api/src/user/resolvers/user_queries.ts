import { Stores } from "../../schema/stores";
import { Context } from "../../context";
import * as GitHubApi from "@octokit/rest";

export function UserQueries(stores: Stores) {
  return {
    async userInfo(root: any, args: any, context: Context) {
      const user = {
        avatarUrl: "unknown",
        username: "unknown",
      };

      if (context.sessionType() === "github") {
        const github = new GitHubApi();
        github.authenticate({
          type: "token",
          token: context.getGitHubToken(),
        });

        const githubUser = await github.users.get({});
        user.avatarUrl = githubUser.data.avatar_url;
        user.username = githubUser.data.login;
      } else if (context.sessionType() === "ship") {
        user.avatarUrl = "";
        user.username = "test";
      }

      return user;
    }
  }
}


