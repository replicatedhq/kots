import * as GitHubApi from "@octokit/rest";
import * as jaeger from "jaeger-client";
import * as _ from "lodash";
import {
  GetNotificationQueryArgs,
  ListNotificationsQueryArgs,
  Notification,
  PullRequestHistoryItem,
  PullRequestHistoryQueryArgs,
} from "../../generated/types";
import { Watch } from "../../watch/";
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
      const watch: Watch = await stores.watchStore.findUserWatch(context.session.userId, { id: args.watchId });
      const notifications = await stores.notificationStore.listNotifications(watch.id!);
      return notifications;
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
