import * as GitHubApi from "@octokit/rest";
import * as jaeger from "jaeger-client";
import * as _ from "lodash";
import { instrumented } from "monkit";
import { Service } from "ts-express-decorators";
import { authorized } from "../auth/decorators";
import {
  CreateFirstPullRequestMutationArgs,
  CreateNotificationMutationArgs,
  DeleteNotificationMutationArgs,
  EnableNotificationMutationArgs,
  GetNotificationQueryArgs,
  ListNotificationsQueryArgs,
  Notification,
  PullRequestHistoryItem,
  PullRequestHistoryQueryArgs,
  UpdateNotificationMutationArgs,
  UpdatePullRequestHistoryMutationArgs,
  WatchItem,
} from "../generated/types";
import { Mutation, Query } from "../schema/decorators";
import { Context } from "../server/server";
import { tracer } from "../server/tracing";
import { WatchStore } from "../watch/watch_store";
import { NotificationStore } from "./store";
import { QueryDocumentKeys } from "graphql/language/visitor";

@Service()
export class ShipNotification {
  constructor(private readonly notificationStore: NotificationStore, private readonly watchStore: WatchStore) {}

  @Mutation("ship-cloud")
  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async updatePullRequestHistory(root: any, args: UpdatePullRequestHistoryMutationArgs, context: Context): Promise<PullRequestHistoryItem[]> {
    const span: jaeger.SpanContext = tracer().startSpan("query.updatePullRequestHistory");

    const notification = await this.notificationStore.findUserNotification(span.context(), context.userId, args.notificationId);
    const result = await this.notificationStore.listPullRequestHistory(span.context(), notification.id!);

    // This probably should use a webhook implementation from github eventually to cache and store the results
    // in our database...  but for now...
    // Also this only returns the status for the latest 100...  This is just enough to validate that
    // the UI can display.
    const github = new GitHubApi();
    github.authenticate({
      type: "token",
      token: context.auth,
    });

    const params: GitHubApi.PullRequestsGetAllParams = {
      owner: notification.pullRequest!.org,
      repo: notification.pullRequest!.repo,
      state: "all",
      base: notification.pullRequest!.branch || "master",
      sort: "created",
      direction: "desc",
      per_page: 100,
    };
    const { data: pullRequestData } = await github.pullRequests.getAll(params);
    await Promise.all(
      result.map(async version => {
        //tslint:disable-next-line underscore-consistent-invocation
        const pullRequest = _.find(pullRequestData, { number: version.number! });
        if (pullRequest && pullRequest.state) {
          let merged = false;
          if (pullRequest.state === "closed") {
            const prParams: GitHubApi.PullRequestsCheckMergedParams = {
              owner: notification.pullRequest!.org,
              repo: notification.pullRequest!.repo,
              number: pullRequest.number!,
            };
            try {
              const githubResponse = await github.pullRequests.checkMerged(prParams);
              merged = githubResponse.status === 204;
            } catch {
              merged = false;
            }
          }
          await this.notificationStore.updatePullRequestStatus(span.context(), params.owner, params.repo, pullRequest.number!, pullRequest.state, merged);
        }
      }),
    );

    const newResult = await this.notificationStore.listPullRequestHistory(span.context(), notification.id!);

    span.finish();

    return newResult;
  }

  @Query("ship-cloud")
  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async pullRequestHistory(root: any, args: PullRequestHistoryQueryArgs, context: Context): Promise<PullRequestHistoryItem[]> {
    const span: jaeger.SpanContext = tracer().startSpan("query.pullRequestHistory");

    const notification = await this.notificationStore.findUserNotification(span.context(), context.userId, args.notificationId);
    const result = await this.notificationStore.listPullRequestHistory(span.context(), notification.id!);

    span.finish();

    return result;
  }

  @Query("ship-cloud")
  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async listNotifications(root: any, args: ListNotificationsQueryArgs, context: Context): Promise<Notification[]> {
    const span: jaeger.SpanContext = tracer().startSpan("query.getNotifications");

    const watch: WatchItem = await this.watchStore.findUserWatch(span.context(), context.userId, { id: args.watchId });
    const notifications = await this.notificationStore.listNotifications(span.context(), watch.id!);
    // Not surfacing to UI currently, but optional
    const pendingNotifications = await this.notificationStore.listPendingPRNotifications(span.context(), watch.id!);
    const allNotifications = _.concat(notifications, pendingNotifications);

    const github = new GitHubApi();
    github.authenticate({
      type: "token",
      token: context.auth,
    });

    // Update PR history
    allNotifications.map(async notification => {
      if (notification.pullRequest) {
        const historyItems = await this.notificationStore.listPullRequestHistory(span.context(), notification.id!);
        historyItems.map(async historyItem => {
          if (historyItem.status === "unknown" || historyItem.status === "pending") {
            const params: GitHubApi.PullRequestsGetParams = {
              owner: notification.pullRequest!.org,
              repo: notification.pullRequest!.repo,
              number: historyItem.number!,
            };
            const { data: pullRequestData } = await github.pullRequests.get(params);
            let merged = false;
            if (pullRequestData.state === "closed") {
              merged = pullRequestData.merged === true;
            }
            await this.notificationStore.updatePullRequestStatus(
              span.context(),
              params.owner,
              params.repo,
              params.number,
              pullRequestData.state!,
              merged,
            );
          }
        });
      }
    })

    span.finish();

    return allNotifications;
  }

  @Query("ship-cloud")
  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async getNotification(root: any, args: GetNotificationQueryArgs, context: Context): Promise<Notification> {
    const span: jaeger.SpanContext = tracer().startSpan("query.getNotifications");

    const notification = await this.notificationStore.findUserNotification(span.context(), context.userId, args.notificationId);
    const result = await this.notificationStore.getNotification(span.context(), notification.id!);

    span.finish();

    return result;
  }

  @Mutation("ship-cloud")
  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async createFirstPullRequest(root: any, args: CreateFirstPullRequestMutationArgs, context: Context): Promise<number> {
    const span: jaeger.SpanContext = tracer().startSpan("mutation.createFirstPullReqeust");

    const watch: WatchItem = await this.watchStore.findUserWatch(span.context(), context.userId, { id: args.watchId });
    const currentState = await this.watchStore.getStateJSON(span.context(), watch.id!);

    let versionLabel: string = "";
    try {
      versionLabel = currentState.v1.metadata.version;
    } catch {
      // ignore
    }

    const notificationId = args.notificationId || "";

    const result = await this.notificationStore.createFirstPullRequest(
      span.context(),
      context.metadata![args.pullRequest!.org.toLowerCase()],
      watch.id!,
      notificationId,
      versionLabel,
      args.pullRequest!,
    );

    span.finish();

    return result;
  }

  @Mutation("ship-cloud")
  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async createNotification(root: any, args: CreateNotificationMutationArgs, context: Context): Promise<Notification> {
    const span: jaeger.SpanContext = tracer().startSpan("mutation.createNotification");

    const { watchId, webhook, email } = args;

    const watch: WatchItem = await this.watchStore.findUserWatch(span.context(), context.userId, { id: watchId });

    const result = await this.notificationStore.createNotification(span.context(), watch.id!, false, webhook || undefined, email || undefined);

    span.finish();

    return result;
  }

  @Mutation("ship-cloud")
  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async enableNotification(root: any, args: EnableNotificationMutationArgs, context: Context): Promise<Notification> {
    const span: jaeger.SpanContext = tracer().startSpan("mutation.toggleEnableNotification");

    const { watchId, notificationId, enabled } = args;

    // This will exception if not authorized
    await this.watchStore.findUserWatch(span.context(), context.userId, { id: watchId });

    await this.notificationStore.toggleEnableNotification(span.context(), notificationId, enabled);

    span.finish();

    return this.getNotification(root, { notificationId }, context);
  }

  @Mutation("ship-cloud")
  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async updateNotification(root: any, args: UpdateNotificationMutationArgs, context: Context): Promise<Notification> {
    const span: jaeger.SpanContext = tracer().startSpan("mutation.updateNotification");

    const { watchId, notificationId, webhook, email } = args;

    // This will throw if not authorized
    await this.watchStore.findUserWatch(span.context(), context.userId, { id: watchId });

    const result = await this.notificationStore.updateNotification(span.context(), notificationId, webhook || undefined, email || undefined);

    span.finish();

    return result;
  }

  @Mutation("ship-cloud")
  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async deleteNotification(root: any, args: DeleteNotificationMutationArgs, context: Context) {
    const span: jaeger.SpanContext = tracer().startSpan("mutation.deleteNotificaton");

    if(args.isPending) {
      const notification = await this.notificationStore.findPendingUserNotification(span.context(), context.userId, args.id);
      await this.notificationStore.deletePendingNotificationById(span.context(), notification.id!);
    } else {
      const notification = await this.notificationStore.findUserNotification(span.context(), context.userId, args.id);
      await this.notificationStore.deleteNotification(span.context(), notification.id!);
    }

    span.finish();
    return true;
  }


}
