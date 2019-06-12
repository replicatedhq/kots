import * as _ from "lodash";
import {
  Notification,
  PullRequestHistoryItem,
} from "../../generated/types";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { ReplicatedError } from "../../server/errors";

export function NotificationQueries(stores: Stores) {
  return {
    async pullRequestHistory(root: any, args: any, context: Context): Promise<PullRequestHistoryItem[]> {
      const notification = await context.getNotification(args.notificationId);
      if (!notification) {
        throw new ReplicatedError("Notification not found");
      }

      const result = await stores.notificationStore.listPullRequestHistory(notification.id!);

      return result;
    },

    async listNotifications(root: any, args: any, context: Context): Promise<Notification[]> {
      const watch = await context.getWatch(args.watchId);
      const notifications = await stores.notificationStore.listNotifications(watch.id);
      return notifications;
    },

    async getNotification(root: any, args: any, context: Context): Promise<Notification> {
      const notification = await context.getNotification(args.notificationId);
      if (!notification) {
        throw new ReplicatedError("Notification not found");
      }

      return notification;
    },


  }
}
