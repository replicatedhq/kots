import * as GitHubApi from "@octokit/rest";
import * as jaeger from "jaeger-client";
import * as _ from "lodash";
import {
  GetNotificationQueryArgs,
  ListNotificationsQueryArgs,
  Notification,
  PullRequestHistoryItem,
  PullRequestHistoryQueryArgs,
  WatchItem,
} from "../../generated/types";
import { Context } from "../../context";
import { tracer } from "../../server/tracing";
import { Stores } from "../../schema/stores";

export function NotificationQueries(stores: Stores) {
  return {
    async pullRequestHistory(root: any, args: PullRequestHistoryQueryArgs, context: Context): Promise<PullRequestHistoryItem[]> {
      const span: jaeger.SpanContext = tracer().startSpan("query.pullRequestHistory");

      const notification = await stores.notificationStore.findUserNotification(span.context(), context.session.userId, args.notificationId);
      const result = await stores.notificationStore.listPullRequestHistory(span.context(), notification.id!);

      span.finish();

      return result;
    },

    async listNotifications(root: any, args: ListNotificationsQueryArgs, context: Context): Promise<Notification[]> {
      const span: jaeger.SpanContext = tracer().startSpan("query.getNotifications");

      const watch: WatchItem = await stores.watchStore.findUserWatch(context.session.userId, { id: args.watchId });
      const notifications = await stores.notificationStore.listNotifications(span.context(), watch.id!);
      // Not surfacing to UI currently, but optional
      const pendingNotifications = await stores.notificationStore.listPendingPRNotifications(span.context(), watch.id!);
      const allNotifications = _.concat(notifications, pendingNotifications);

      const github = new GitHubApi();
      github.authenticate({
        type: "token",
        token: context.getGitHubToken(),
      });

      // Update PR history
      allNotifications.map(async notification => {
        if (notification.pullRequest) {
          const historyItems = await stores.notificationStore.listPullRequestHistory(span.context(), notification.id!);
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
              await stores.notificationStore.updatePullRequestStatus(
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
    },

    async getNotification(root: any, args: GetNotificationQueryArgs, context: Context): Promise<Notification> {
      const span: jaeger.SpanContext = tracer().startSpan("query.getNotifications");

      const notification = await stores.notificationStore.findUserNotification(span.context(), context.session.userId, args.notificationId);
      const result = await stores.notificationStore.getNotification(notification.id!);

      span.finish();

      return result;
    },


  }
}
