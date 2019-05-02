import * as Express from "express";
import * as jaeger from "jaeger-client";
import { DataDogMetricRegistry, instrumented } from "monkit";
import { BodyParams, Controller, HeaderParams, Post, Req, Res } from "ts-express-decorators";
import { NotificationStore } from "../notification/store";
import { logger } from "../server/logger";
import { traced, tracer } from "../server/tracing";
import { ClusterStore } from "../cluster/cluster_store";
import { WatchStore } from "../watch/watch_store";

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
}

interface ErrorResponse {
  error: {};
}

/**
 *  gets hooks from github
 */
@Controller("api/v1/hooks/github")
export class GitHubHookAPI {
  constructor(
    private readonly notificationStore: NotificationStore,
    private readonly clusterStore: ClusterStore,
    private readonly watchStore: WatchStore,
    private readonly metrics: DataDogMetricRegistry,
  ) {
  }

  @Post("")
  @instrumented({tags: ["tier:webhook"]})
  async githubHook(
    @Res() response: Express.Response,
    @Req() request: Express.Request,
    @HeaderParams("x-github-event") eventType: string,
    @BodyParams("") body?: { action?: string }, // we're just gonna cast this later
  ): Promise<{} | ErrorResponse> {
    const span: jaeger.SpanContext = tracer().startSpan("githubHookAPI.githubHook");
    span.setTag("eventType", eventType);

    logger.info(`received github hook for eventType ${eventType}`);

    const action = body && body.action;
    this.metrics.meter(`GitHubHookAPI.events`, [`eventType:${eventType}`, `action:${action}`]).mark();

    switch (eventType) {
      case "pull_request": {
        await this.handlePullRequest(span.context(), body as GitHubPullRequestEvent);
        return this.success(span, response);
      }

      case "installation": {
        await this.handleInstallation(span.context(), body as GitHubInstallationRequest);
        return this.success(span, response);
      }

      // ignored
      case "installation_repositories":
      case "integration_installation":
      case "integration_installation_repositories":
        return this.success(span, response);

      // default is an error
      default: {
        logger.info(`Unexpected event type in GitHub hook: ${eventType}`);
        response.status(204);
        span.finish();
        return {};
      }
    }
  }

  private success(span: jaeger.SpanContext, response: Express.Response): {} {
    span.finish();
    response.status(204);
    return {};
  }

  @instrumented()
  @traced()
  private async handleInstallation(ctx: jaeger.SpanContext, installationEvent: GitHubInstallationRequest): Promise<void> {
    if (installationEvent.action === "created") {
      // Do we care?
    } else if (installationEvent.action === "deleted") {
      // Should we delete all pullrequest notifications?
    }
  }

  @instrumented()
  @traced()
  private async handlePullRequest(ctx: jaeger.SpanContext, pullRequestEvent: GitHubPullRequestEvent): Promise<void> {
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

    const clusters = await this.clusterStore.listClustersForGitHubRepo(ctx, owner, repo);

    for (const cluster of clusters) {
      const watches = await this.watchStore.listForCluster(ctx, cluster.id!);

      for (const watch of watches) {
        const pendingVersions = await this.watchStore.listPendingVersions(ctx, watch.id!);
        for (const pendingVersion of pendingVersions) {
          if (pendingVersion.pullrequestNumber === pullRequestEvent.number) {
            await this.watchStore.updateVersionStatus(ctx, watch.id!, pendingVersion.sequence!, status);
            if (pullRequestEvent.pull_request.merged) {
              await this.watchStore.setCurrentVersion(ctx, watch.id!, pendingVersion.sequence!);
            }
            return;
          }
        }

        const pastVersions = await this.watchStore.listPastVersions(ctx, watch.id!);
        for (const pastVersion of pastVersions) {
          if (pastVersion.pullrequestNumber === pullRequestEvent.number) {
            await this.watchStore.updateVersionStatus(ctx, watch.id!, pastVersion.sequence!, status);
            if (pullRequestEvent.pull_request.merged) {
              await this.watchStore.setCurrentVersion(ctx, watch.id!, pastVersion .sequence!);
            }
            return;
          }
        }

      }
    }

    logger.warn(`received unhandled github pull request event`);
  }
}
