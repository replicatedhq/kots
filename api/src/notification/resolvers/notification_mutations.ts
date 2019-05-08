import * as GitHubApi from "@octokit/rest";
import * as jaeger from "jaeger-client";
import * as _ from "lodash";
import {
  CreateFirstPullRequestMutationArgs,
  CreateNotificationMutationArgs,
  DeleteNotificationMutationArgs,
  EnableNotificationMutationArgs,
  Notification,
  PullRequestHistoryItem,
  UpdateNotificationMutationArgs,
  UpdatePullRequestHistoryMutationArgs,
  WatchItem,
} from "../../generated/types";
import { Context } from "../../context";
import { tracer } from "../../server/tracing";

export function NotificationMutations(stores: any) {
  return {
    async updatePullRequestHistory(root: any, args: UpdatePullRequestHistoryMutationArgs, context: Context): Promise<PullRequestHistoryItem[]> {
      const span: jaeger.SpanContext = tracer().startSpan("query.updatePullRequestHistory");

      const notification = await stores.notificationStore.findUserNotification(span.context(), context.session.userId, args.notificationId);
      const result = await stores.notificationStore.listPullRequestHistory(span.context(), notification.id!);

      // This probably should use a webhook implementation from github eventually to cache and store the results
      // in our database...  but for now...
      // Also this only returns the status for the latest 100...  This is just enough to validate that
      // the UI can display.
      const github = new GitHubApi();
      github.authenticate({
        type: "token",
        token: context.getGitHubToken(),
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
            await stores.notificationStore.updatePullRequestStatus(span.context(), params.owner, params.repo, pullRequest.number!, pullRequest.state, merged);
          }
        }),
      );

      const newResult = await stores.notificationStore.listPullRequestHistory(span.context(), notification.id!);

      span.finish();

      return newResult;
    },

    async createFirstPullRequest(root: any, args: CreateFirstPullRequestMutationArgs, context: Context): Promise<number> {
      const span: jaeger.SpanContext = tracer().startSpan("mutation.createFirstPullReqeust");

      const watch: WatchItem = await stores.watchStore.findUserWatch(span.context(), context.session.userId, { id: args.watchId });
      const currentState = await stores.watchStore.getStateJSON(span.context(), watch.id!);

      let versionLabel: string = "";
      try {
        versionLabel = currentState.v1.metadata.version;
      } catch {
        // ignore
      }

      const notificationId = args.notificationId || "";

      const result = await stores.notificationStore.createFirstPullRequest(
        span.context(),
        context.session.metadata[args.pullRequest!.org.toLowerCase()],
        watch.id!,
        notificationId,
        versionLabel,
        args.pullRequest!,
      );

      span.finish();

      return result;
    },

    async createNotification(root: any, args: CreateNotificationMutationArgs, context: Context): Promise<Notification> {
      const span: jaeger.SpanContext = tracer().startSpan("mutation.createNotification");

      const { watchId, webhook, email } = args;

      const watch: WatchItem = await stores.watchStore.findUserWatch(span.context(), context.session.userId, { id: watchId });

      const result = await stores.notificationStore.createNotification(span.context(), watch.id!, false, webhook || undefined, email || undefined);

      span.finish();

      return result;
    },

    async enableNotification(root: any, args: EnableNotificationMutationArgs, context: Context): Promise<Notification> {
      const span: jaeger.SpanContext = tracer().startSpan("mutation.toggleEnableNotification");

      const { watchId, notificationId, enabled } = args;

      // This will exception if not authorized
      await stores.watchStore.findUserWatch(span.context(), context.session.userId, { id: watchId });

      await stores.notificationStore.toggleEnableNotification(span.context(), notificationId, enabled);

      span.finish();

      return this.getNotification(root, { notificationId }, context);
    },

    async updateNotification(root: any, args: UpdateNotificationMutationArgs, context: Context): Promise<Notification> {
      const span: jaeger.SpanContext = tracer().startSpan("mutation.updateNotification");

      const { watchId, notificationId, webhook, email } = args;

      // This will throw if not authorized
      await stores.watchStore.findUserWatch(span.context(), context.session.userId, { id: watchId });

      const result = await stores.notificationStore.updateNotification(span.context(), notificationId, webhook || undefined, email || undefined);

      span.finish();

      return result;
    },

    async deleteNotification(root: any, args: DeleteNotificationMutationArgs, context: Context) {
      const span: jaeger.SpanContext = tracer().startSpan("mutation.deleteNotificaton");

      if(args.isPending) {
        const notification = await stores.notificationStore.findPendingUserNotification(span.context(), context.session.userId, args.id);
        await stores.notificationStore.deletePendingNotificationById(span.context(), notification.id!);
      } else {
        const notification = await stores.notificationStore.findUserNotification(span.context(), context.session.userId, args.id);
        await stores.notificationStore.deleteNotification(span.context(), notification.id!);
      }

      span.finish();
      return true;
    }
  }
}
