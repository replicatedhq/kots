import * as _ from "lodash";
import {
  Notification,
  UpdateNotificationMutationArgs,
} from "../../generated/types";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { ReplicatedError } from "../../server/errors";

export function NotificationMutations(stores: Stores) {
  return {
    async createNotification(root: any, args: any, context: Context): Promise<Notification> {
      const watch = await context.getWatch(args.watchId);

      const webhookNotification = args.webhook;
      const emailNotification = args.email;

      const result = await stores.notificationStore.createNotification(
        watch.id,
        webhookNotification ? webhookNotification.uri : undefined,
        emailNotification ? emailNotification.recipientAddress : undefined,
      );

      return result;
    },

    async enableNotification(root: any, args: any, context: Context): Promise<Notification> {
      const watch = await context.getWatch(args.watchId);
      const notifications = await stores.notificationStore.listNotifications(watch.id);
      const notification = _.find(notifications, { id: args.notificationId });
      if (!notification) {
        throw new ReplicatedError("Notification not found");
      }

      await stores.notificationStore.toggleEnableNotification(notification.id!, args.enabled);

      return this.getNotification(root, notification.id, context);
    },

    async updateNotification(root: any, args: UpdateNotificationMutationArgs, context: Context): Promise<Notification> {
      const watch = await context.getWatch(args.watchId);
      const notifications = await stores.notificationStore.listNotifications(watch.id);
      const notification = _.find(notifications, { id: args.notificationId });
      if (!notification) {
        throw new ReplicatedError("Notification not found");
      }

      const webhookNotification = args.webhook;
      const emailNotification = args.email;

      const result = await stores.notificationStore.updateNotification(
        notification.id!,
        webhookNotification ? webhookNotification.uri! : undefined,
        emailNotification ? emailNotification.recipientAddress! : undefined
      );

      return result;
    },

    async deleteNotification(root: any, args: any, context: Context) {
      const watch = await context.getWatch(args.watchId);
      const notifications = await stores.notificationStore.listNotifications(watch.id);
      const notification = _.find(notifications, { id: args.notificationId });
      if (!notification) {
        throw new ReplicatedError("Notification not found");
      }

      await stores.notificationStore.deleteNotification(notification.id!);
      return true;
    }
  }
}
