import * as Express from "express";
import { BodyParams, Controller, HeaderParams, Post, Req, Res } from "ts-express-decorators";
import { logger } from "../server/logger";

interface GitHubInstallationRequest {
  action: string;
  installation: {
    id: number;
    access_tokens_url: string;
    account: {
      login: string;
      id: number;
      // tslint:disable-next-line no-reserved-keywords
      type: string;
    };
  };
  sender: {
    login: string;
    id: number;
  };
}

interface GitHubPullRequestEvent {
  action: string;
  //tslint:disable-next-line no-reserved-keywords
  number: number;
  pull_request: {
    state: string;
    merged: boolean;
    base: {
      repo: {
        name: string;
        owner: {
          login: string;
        };
      };
    };
  };
  merged_at: string;
}

interface ErrorResponse {
  error: {};
}

/**
 *  gets hooks from github
 */
@Controller("api/v1/hooks/github")
export class GitHubHookAPI {
  @Post("")
  async githubHook(
    @Res() response: Express.Response,
    @Req() request: Express.Request,
    @HeaderParams("x-github-event") eventType: string,
    @BodyParams("") body?: { action?: string }, // we're just gonna cast this later
  ): Promise<{} | ErrorResponse> {
    logger.info({msg: `received github hook for eventType ${eventType}`});

    const action = body && body.action;

    switch (eventType) {
      case "pull_request": {
        await this.handlePullRequest(request, body as GitHubPullRequestEvent);
        response.status(204);
        return {};
      }

      case "installation": {
        await this.handleInstallation(body as GitHubInstallationRequest);
        response.status(204);
        return {};
      }

      // ignored
      case "installation_repositories":
      case "integration_installation":
      case "integration_installation_repositories":
        response.status(204);
        return {};

      // default is an error
      default: {
        logger.info({msg: `Unexpected event type in GitHub hook: ${eventType}`});
        response.status(204);
        return {};
      }
    }
  }

  private async handleInstallation(installationEvent: GitHubInstallationRequest): Promise<void> {
    if (installationEvent.action === "created") {
      // Do we care?
    } else if (installationEvent.action === "deleted") {
      // Should we delete all pullrequest notifications?
    }
  }

  private async handlePullRequest(request: Express.Request, pullRequestEvent: GitHubPullRequestEvent): Promise<void> {
    const handledActions: string[] = ["opened", "closed", "reopened"];
    if (handledActions.indexOf(pullRequestEvent.action) === -1) {
      return;
    }

    let status = pullRequestEvent.action;
    if (pullRequestEvent.pull_request.merged) {
      status = "merged";
    }

    const owner = pullRequestEvent.pull_request.base.repo.owner.login;
    const repo = pullRequestEvent.pull_request.base.repo.name;

    const clusters = await request.app.locals.stores.clusterStore.listClustersForGitHubRepo(owner, repo);

    for (const cluster of clusters) {
      const watches = await request.app.locals.stores.watchStore.listForCluster(cluster.id!);
      for (const watch of watches) {
        const pendingVersions = await request.app.locals.stores.watchStore.listPendingVersions(watch.id!);
        for (const pendingVersion of pendingVersions) {
          if (pendingVersion.pullrequestNumber === pullRequestEvent.number) {
            await request.app.locals.stores.watchStore.updateVersionStatus(watch.id!, pendingVersion.sequence!, status);
            if (pullRequestEvent.pull_request.merged) {
              // When a pull request closes multiple commits, the order in which hooks come in is random. 
              // We should not update the current ssequence to something lower than what it already is.
              // This will create a bug where we show a PR as not merged but GH will show it as merged
              // because they automatically do it. This will be fixed when we verify commit sha's on our end.
              if (pendingVersion.sequence! < watch.currentVersion.sequence!) {
                return;
              }
              await request.app.locals.stores.watchStore.setCurrentVersion(watch.id!, pendingVersion.sequence!, pullRequestEvent.merged_at);
            }
            return;
          }
        }

        const pastVersions = await request.app.locals.stores.watchStore.listPastVersions(watch.id!);
        for (const pastVersion of pastVersions) {
          if (pastVersion.pullrequestNumber === pullRequestEvent.number) {
            await request.app.locals.stores.watchStore.updateVersionStatus(watch.id!, pastVersion.sequence!, status);
            if (pullRequestEvent.pull_request.merged) {
              await request.app.locals.stores.watchStore.setCurrentVersion(watch.id!, pastVersion .sequence!, pullRequestEvent.merged_at);
            }
            return;
          }
        }

      }
    }

    logger.warn({msg: `received unhandled github pull request event`});
  }
}
