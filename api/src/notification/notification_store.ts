import randomstring from "randomstring";
import {
  Notification,
  PullRequestHistoryItem,
} from "../generated/types";
import { ReplicatedError } from "../server/errors";
import pg from "pg";

export class NotificationStore {
  constructor(private readonly pool: pg.Pool) {}

  async listPullRequestHistory(notificationId: string): Promise<PullRequestHistoryItem[]> {
    const q = `select version_label, pullrequest_number, org, repo, branch, root_path, created_at,
      github_status, sequence, source_branch from pullrequest_history where notification_id = $1 order by created_at desc`;
    const v = [notificationId];

    const result = await this.pool.query(q, v);

    const pullRequestHistoryItems: PullRequestHistoryItem[] = [];
    for (const row of result.rows) {
      const pullRequestHistoryItem = {
        title: row.version_label !== "" && row.version_label !== "Unknown" ? `Version ${row.version_label}` : "",
        createdOn: row.created_at,
        number: row.pullrequest_number,
        uri: `https://github.com/${row.org}/${row.repo}/pull/${row.pullrequest_number}`,
        status: row.github_status,
        sequence: row.sequence,
        sourceBranch: row.source_branch,
      };

      pullRequestHistoryItems.push(pullRequestHistoryItem);
    }

    return pullRequestHistoryItems;
  }

  async createNotification(watchId: string, webhookUri?: string, recipientAddress?: string): Promise<Notification> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      const q = "insert into ship_notification (id, watch_id, created_at, updated_at, enabled) values ($1, $2, $3, $4, 1)";
      const v = [
        id,
        watchId,
        new Date(),
        null
      ];
      await pg.query(q, v);

      if (webhookUri) {
        const qq = "insert into webhook_notification (notification_id, destination_uri, created_at) values ($1, $2, $3)";
        const vv = [
          id,
          webhookUri,
          new Date()
        ];
        await pg.query(qq, vv);
      }

      if (recipientAddress) {
        const qq = "insert into email_notification (notification_id, recipient, created_at) values ($1, $2, $3)";
        const vv = [
          id,
          recipientAddress,
          new Date()
        ];
        await pg.query(qq, vv);
      }

      await pg.query("commit");

      return this.getNotification(id);
    } catch (err) {
      await pg.query("rollback");
      throw err;
    } finally {
      pg.release();
    }
  }

  async updateNotification(id: string, webhookUri?: string, recipientAddress?: string): Promise<Notification> {
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      const q = "update ship_notification set updated_at = $2 where id = $1";
      const v = [id, new Date()];
      await pg.query(q, v);

      if (webhookUri) {
        const qq = "update webhook_notification set destination_uri = $2 where notification_id = $1";
        const vv = [
          id,
          webhookUri
        ];
        await pg.query(qq, vv);
      }

      if (recipientAddress) {
        const qq = "update email_notification set recipient = $2 where notification_id = $1";
        const vv = [
          id,
          recipientAddress
        ];
        await pg.query(qq, vv);
      }

      await pg.query("commit");

      return this.getNotification(id);
    } catch(err) {
      pg.query("rollback");
      throw err;
    } finally {
      pg.release();
    }
  }

  async toggleEnableNotification(notificationId: string, enabled: number): Promise<void> {
    const q = "update ship_notification set enabled = $2 where id = $1";
    const v = [
      notificationId,
      enabled
    ];

    await this.pool.query(q, v);
    return;
  }

  async getNotification(id: string): Promise<Notification> {
    const q = `select n.id,
            n.watch_id,
            n.created_at,
            n.updated_at,
            n.triggered_at,
            n.enabled,
            whn.destination_uri,
            en.recipient
      from ship_notification n
            left outer join webhook_notification whn on whn.notification_id = n.id
            left outer join email_notification en on en.notification_id = n.id
      where n.id = $1`;
    const v = [id];

    const result = await this.pool.query(q, v);
    const row = result.rows[0];
    const notification: Notification = {
      id: row.id,
      createdOn: row.created_at,
      updatedOn: row.updated_at,
      triggeredOn: row.triggered_at,
      enabled: row.enabled,
      webhook: null,
      email: null,
    };

    if (row.destination_uri) {
      notification.webhook = {
        uri: row.destination_uri,
      };
    }

    if (row.recipient) {
      notification.email = {
        recipientAddress: row.recipient,
      };
    }

    return notification;
  }

  async deleteNotification(notificationId: string): Promise<void> {
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      let q = `delete
              from ship_notification
              where id = $1`;
      let v = [notificationId];
      await pg.query(q, v);

      q = `delete
          from webhook_notification
          where notification_id = $1`;
      v = [notificationId];
      await pg.query(q, v);

      q = `delete
          from email_notification
          where notification_id = $1`;
      v = [notificationId];
      await pg.query(q, v);

      q = `delete
          from pullrequest_notification
          where notification_id = $1`;
      v = [notificationId];
      await pg.query(q, v);

      q = `delete
          from pullrequest_history
          where notification_id = $1`;
      v = [notificationId];
      await pg.query(q, v);

      await pg.query("commit");
    } catch (err) {
      await pg.query("rollback");
      throw err;
    } finally {
      pg.release();
    }
  }

  async findUserNotification(userId: string, notificationId: string): Promise<Notification> {
    const q = `SELECT n.id
              FROM ship_notification n
                      INNER JOIN watch w ON w.id = n.watch_id
                      INNER JOIN user_watch u ON u.watch_id = w.id
              WHERE n.id = $1
                AND u.user_id = $2`;
    const v = [notificationId, userId];

    const { rows }: { rows: any[] } = await this.pool.query(q, v);
    if (rows.length === 0) {
      throw new ReplicatedError("Notification not found");
    }

    return this.getNotification(rows[0].id);
  }

  async listNotifications(watchId: string): Promise<Notification[]> {
    const q = `select n.id from ship_notification n
    inner join watch on watch.id = n.watch_id
    where watch.id = $1 order by n.created_at`;
    const v = [watchId];

    const result = await this.pool.query(q, v);
    const notifications: Notification[] = [];
    for (const row of result.rows) {
      const notification = await this.getNotification(row.id);
      notifications.push(notification);
    }
    return notifications;
  }

  async createShipNotification(notificationID: string, watchID: string, createdAt: any): Promise<void> {
    const q = `insert into ship_notification
      (id, watch_id, created_at, enabled)
      values
      ($1, $2, $3, $4)`;
    const v = [notificationID, watchID, createdAt, 1];
    await this.pool.query(q, v);
  }
}
