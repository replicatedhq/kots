import * as jaeger from "jaeger-client";
import { each } from "lodash";
import { instrumented } from "monkit";
import { QueryResult } from "pg";
import * as randomstring from "randomstring";
import * as rp from "request-promise";
import { Service } from "ts-express-decorators";
import {
  EmailNotification,
  EmailNotificationInput,
  Notification,
  PullRequestHistoryItem,
  PullRequestNotification,
  PullRequestNotificationInput,
  WebhookNotificationInput,
} from "../generated/types";
import { ReplicatedError } from "../server/errors";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import { tracer } from "../server/tracing";
import { PostgresWrapper, transaction } from "../util/persistence/db";
import { checkExists, signGetRequest } from "../util/s3";

@Service()
export class NotificationStore {
  constructor(private readonly wrapper: PostgresWrapper, private readonly params: Params) {}

  @instrumented()
  async listPullRequestHistory(ctx: jaeger.SpanContext, notificationId: string): Promise<PullRequestHistoryItem[]> {

    const q = `SELECT version_label, pullrequest_number, org, repo, branch, root_path, created_at, github_status, sequence, source_branch FROM pullrequest_history WHERE notification_id = $1 ORDER BY created_at desc`;
    const v = [notificationId];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);

    const pullRequestHistoryItems: PullRequestHistoryItem[] = [];
    for (const row of rows) {
      const result = this.mapPullRequestHistoryItem(row);
      pullRequestHistoryItems.push(result);
    }

    return pullRequestHistoryItems;
  }

  // this is reallll long
  @instrumented()
  async createFirstPullRequest(ctx: jaeger.SpanContext, githubInstallationID: string, watchId: string, notificationId: string, versionLabel: string, pullRequest: PullRequestNotification): Promise<number> {
    logger.info("githubInstallationID", githubInstallationID);
    logger.info("pullRequest.org", pullRequest.org);

    const q = `select filepath
               from ship_output_files
               where watch_id = $1
               order by sequence desc`;
    const v = [watchId];
    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    let downloadFilepath: string = "";

    // tslint:disable-next-line
    for (let i = 0; i < rows.length; i++) {
      downloadFilepath = rows[i].filepath;

      let bucket: string = "";
      let key: string = "";
      if (process.env.USE_EC2_PARAMETERS) {
        bucket = this.params.shipOutputBucket;
        key = downloadFilepath;
      } else {
        bucket = "s3";
        key = `${this.params.shipOutputBucket.trim()}/${downloadFilepath}`;
      }

      if (await checkExists(this.params, { Bucket: bucket, Key: key })) {
        logger.info(`download filepath: ${downloadFilepath}`);
        break;
      }
    }

    let downloadURL: string = "";
    if (process.env.USE_EC2_PARAMETERS) {
      // Use a presigned url so we can stream to the worker
      downloadURL = await signGetRequest({
        Bucket: this.params.shipOutputBucket.trim(),
        Key: downloadFilepath,
        Expires: 60 * 5,
      });
    } else {
      // tslint:disable-next-line:no-http-string
      downloadURL = `http://s3.default.svc.cluster.local:4569/${this.params.shipOutputBucket.trim()}/${downloadFilepath}`;
    }

    let uri =
      `${this.params.shipWatchBaseURL}/v1/create/pullrequest` +
      `?githubInstallationID=${githubInstallationID}` +
      `&org=${encodeURIComponent(pullRequest.org)}` +
      `&repo=${encodeURIComponent(pullRequest.repo)}` +
      `&branch=${encodeURIComponent(pullRequest.branch!)}` +
      `&rootPath=${encodeURIComponent(pullRequest.rootPath!)}` +
      `&watchID=${watchId}` +
      `&versionLabel=${encodeURIComponent(versionLabel)}`;

    if (notificationId !== "") {
      uri = `${uri}&existingID=${notificationId}`
    }

    const options = {
      method: "POST",
      uri,
      headers: {
        "X-TraceContext": ctx,
      },
      formData: {
        "output.tar.gz": rp(downloadURL),
      },
    };

    const body = await rp(options);
    const parsedBody = JSON.parse(body);

    const pendingPRQuery = `insert into pending_pullrequest_notification
      (pullrequest_history_id, watch_id, org, repo, branch, root_path, created_at, github_installation_id, pullrequest_number)
      values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`;
    const pendingPRValues = [
      parsedBody.id,
      parsedBody.watch_id,
      pullRequest.org,
      pullRequest.repo,
      pullRequest.branch,
      pullRequest.rootPath,
      new Date(),
      githubInstallationID,
      parsedBody.pr_number
    ];

    await this.wrapper.query(pendingPRQuery, pendingPRValues);


    return parsedBody.pr_number;
  }

  @instrumented()
  async updatePullRequestStatus(ctx: jaeger.SpanContext, owner: string, repo: string, prNumber: number, status: string, merged: boolean): Promise<void> {
    const updatePrHistoryCtx = tracer().startSpan("query.updatePrHistory", {childOf:ctx});
    let shipState: string;

    switch (status) {
      case "open":
        shipState = "pending";
        break;
      case "closed":
        shipState = merged ? "deployed" : "ignored";
        break;
      default:
        shipState = "unknown";
    }

    const q = `update pullrequest_history
               set github_status = $1
               where org = $2
                 and repo = $3
                 and pullrequest_number = $4`;
    const v = [shipState, owner, repo, prNumber];

    await this.wrapper.query(q, v);
    updatePrHistoryCtx.finish();

    if (shipState === "deployed") {
      // check to see if this pr is in the pending_pullrequest_notification table
      // if it is, remove it from that table and create an entry in the pullrequest_notifcation and ship_notification tables
      await transaction(this.wrapper, async (client: PostgresWrapper) => {
        const pendingPrResult = await this.checkPendingPrNotification(updatePrHistoryCtx.context(), client, owner, repo, prNumber)
        if (pendingPrResult.rowCount === 1) {
          // pending pr notification exists!
          // now check if this is an update, rather than a creation
          const existingNotification = await this.getNotification(ctx, pendingPrResult.rows[0].pullrequest_history_id)
          if (existingNotification.id === "") {
            // this is a new notification and needs to be created in both the pullrequest_notification and ship_notification tables
            await this.createPrNotification(updatePrHistoryCtx.context(), client, pendingPrResult.rows[0].pullrequest_history_id, owner, repo, pendingPrResult.rows[0].branch, pendingPrResult.rows[0].root_path, pendingPrResult.rows[0].created_at, pendingPrResult.rows[0].github_installation_id)
            await this.createShipNotification(updatePrHistoryCtx.context(), client, pendingPrResult.rows[0].pullrequest_history_id, pendingPrResult.rows[0].watch_id, pendingPrResult.rows[0].created_at)
          } else {
            // notification already exists, so we just need to update it
            await this.updatePrNotification(updatePrHistoryCtx.context(), client, pendingPrResult.rows[0].pullrequest_history_id, owner, repo, pendingPrResult.rows[0].branch, pendingPrResult.rows[0].root_path, pendingPrResult.rows[0].github_installation_id)
          }
          await this.deletePendingPrNotification(updatePrHistoryCtx.context(), client, owner, repo, prNumber);
        }
      });
    }
  }

  @instrumented()
  async createNotification(ctx: jaeger.SpanContext, watchId: string, isDefault: boolean, webhook?: WebhookNotificationInput, email?: EmailNotificationInput): Promise<Notification> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    const queries: any[] = [];

    const q = "INSERT INTO ship_notification (id, watch_id, created_at, updated_at, enabled) VALUES ($1, $2, $3, $4, 1)";
    const v = [id, watchId, new Date(), isDefault ? null : new Date()];
    queries.push({ query: q, values: v });

    if (webhook) {
      const qq = "INSERT INTO webhook_notification (notification_id, destination_uri, created_at) VALUES ($1, $2, $3)";
      const vv = [id, webhook.uri, new Date()];
      queries.push({ query: qq, values: vv });
    }

    if (email) {
      const qq = "INSERT INTO email_notification (notification_id, recipient, created_at) VALUES ($1, $2, $3)";
      const vv = [id, email.recipientAddress, new Date()];
      queries.push({ query: qq, values: vv });
    }

    await transaction(this.wrapper, async (client: PostgresWrapper) => {
      each(queries, async query => {
        await client.query(query.query, query.values);
      });
    });

    return this.getNotification(ctx, id);
  }

  @instrumented()
  async updateNotification(ctx: jaeger.SpanContext, id: string, webhook?: WebhookNotificationInput, email?: EmailNotification): Promise<Notification> {
    const queries: any[] = [];

    const q = "UPDATE ship_notification SET updated_at = $2 WHERE id = $1";
    const v = [id, new Date()];
    queries.push({ query: q, values: v });

    if (webhook) {
      const qq = "UPDATE webhook_notification SET destination_uri = $2 WHERE notification_id = $1";
      const vv = [id, webhook.uri];
      queries.push({ query: qq, values: vv });
    }

    if (email) {
      const qq = "UPDATE email_notification SET recipient = $2 WHERE notification_id = $1";
      const vv = [id, email.recipientAddress];
      queries.push({ query: qq, values: vv });
    }

    await transaction(this.wrapper, async (client: PostgresWrapper) => {
      each(queries, async query => {
        await client.query(query.query, query.values);
      });
    });

    return this.getNotification(ctx, id);
  }

  @instrumented()
  async toggleEnableNotification(ctx: jaeger.SpanContext, notificationId: string, enabled: number): Promise<void> {
    const q = "UPDATE ship_notification SET enabled = $2 WHERE id = $1";
    const v = [notificationId, enabled];
    await this.wrapper.query(q, v);
    return;
  }

  @instrumented()
  async findUserNotification(ctx: jaeger.SpanContext, userId: string, notificationId: string): Promise<Notification> {
    const q = `SELECT n.id
               FROM ship_notification n
                      INNER JOIN watch w ON w.id = n.watch_id
                      INNER JOIN user_watch u ON u.watch_id = w.id
               WHERE n.id = $1
                 AND u.user_id = $2`;
    const v = [notificationId, userId];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    if (rows.length === 0) {
      throw new ReplicatedError("Notification not found");
    }

    return this.getNotification(ctx, rows[0].id);
  }

  @instrumented()
  async findPendingUserNotification(ctx: jaeger.SpanContext, userId: string, notificationId: string): Promise<Notification> {
    const q = `SELECT n.pullrequest_history_id
               FROM pending_pullrequest_notification n
                      INNER JOIN watch w ON w.id = n.watch_id
                      INNER JOIN user_watch u ON u.watch_id = w.id
               WHERE n.pullrequest_history_id = $1
               AND u.user_id = $2`;
    const v = [notificationId, userId];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    if (rows.length === 0) {
      throw new ReplicatedError("Notification not found");
    }

    return this.getPendingPRNotification(ctx, rows[0].pullrequest_history_id);
  }

  @instrumented()
  async getPendingPRNotification(ctx: jaeger.SpanContext, id: string): Promise<Notification> {
    const q = `
      SELECT  pullrequest_history_id as id,
              org,
              repo,
              branch,
              root_path,
              created_at,
              github_installation_id,
              pullrequest_number,
              watch_id
      FROM pending_pullrequest_notification
      WHERE pullrequest_history_id = $1`;
    const v = [id];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    if (rows.length > 0) {
      return this.mapRow(rows[0]);
    }
    const emptyNotification: Notification = {}
    emptyNotification.id = ""
    return emptyNotification;
  }

  async getInstallationIdForPullRequestNotification(xtx: jaeger.SpanContext, id: string): Promise<number> {
    const q = `select github_installation_id from pullrequest_notification where notification_id = $1`;
    const v = [id];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    return rows[0].github_installation_id;
  }

  @instrumented()
  async getNotification(ctx: jaeger.SpanContext, id: string): Promise<Notification> {
    const q = `
      SELECT n.id,
             n.watch_id,
             n.created_at,
             n.updated_at,
             n.triggered_at,
             n.enabled,
             whn.destination_uri,
             en.recipient,
             prn.org,
             prn.repo,
             prn.branch,
             prn.root_path
      FROM ship_notification n
             LEFT OUTER JOIN webhook_notification whn ON whn.notification_id = n.id
             LEFT OUTER JOIN email_notification en ON en.notification_id = n.id
             LEFT OUTER JOIN pullrequest_notification prn ON prn.notification_id = n.id
      WHERE n.id = $1`;
    const v = [id];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    if (rows.length > 0) {
      return this.mapRow(rows[0]);
    }
    const emptyNotification: Notification = {}
    emptyNotification.id = ""
    return emptyNotification
  }

  @instrumented()
  async deleteNotification(ctx: jaeger.SpanContext, notificationId: string): Promise<void> {
    await this.wrapper.query("begin");

    try {
      let q = `delete
               from ship_notification
               where id = $1`;
      let v = [notificationId];
      await this.wrapper.query(q, v);

      q = `delete
           from webhook_notification
           where notification_id = $1`;
      v = [notificationId];
      await this.wrapper.query(q, v);

      q = `delete
           from email_notification
           where notification_id = $1`;
      v = [notificationId];
      await this.wrapper.query(q, v);

      q = `delete
           from pullrequest_notification
           where notification_id = $1`;
      v = [notificationId];
      await this.wrapper.query(q, v);

      q = `delete
           from pullrequest_history
           where notification_id = $1`;
      v = [notificationId];
      await this.wrapper.query(q, v);

      await this.wrapper.query("commit");
    } catch (err) {
      await this.wrapper.query("rollback");
      throw err;
    }
  }

  @instrumented()
  async deletePendingNotificationById(ctx: jaeger.SpanContext, prNotificationId: string): Promise<void> {
    await this.wrapper.query("begin");

    try {
      const q = `delete
               from pending_pullrequest_notification
               where pullrequest_history_id = $1`;
      const v = [prNotificationId];
      await this.wrapper.query(q, v);
      await this.wrapper.query("commit");
    } catch {
      await this.wrapper.query("rollback");
    }
  }

  @instrumented()
  async listNotifications(ctx: jaeger.SpanContext, watchId: string): Promise<Notification[]> {
    const q = `
      SELECT n.id,
             n.watch_id,
             n.created_at,
             n.updated_at,
             n.triggered_at,
             n.enabled,
             whn.destination_uri,
             en.recipient,
             prn.org,
             prn.repo,
             prn.branch,
             prn.root_path
      FROM ship_notification n
             LEFT OUTER JOIN webhook_notification whn ON whn.notification_id = n.id
             LEFT OUTER JOIN email_notification en ON en.notification_id = n.id
             LEFT OUTER JOIN pullrequest_notification prn ON prn.notification_id = n.id
             INNER JOIN watch ON watch.id = n.watch_id
      WHERE watch.id = $1
      ORDER BY n.created_at`;
    const v = [watchId];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    const notifications: Notification[] = [];
    for (const row of rows) {
      const result = this.mapRow(row);
      notifications.push(result);
    }
    return notifications;
  }

  @instrumented()
  async listPendingPRNotifications(ctx: jaeger.SpanContext, watchId: string): Promise<Notification[]> {
    const q = `
      SELECT  pullrequest_history_id as id,
              org,
              repo,
              branch,
              root_path,
              created_at,
              github_installation_id,
              pullrequest_number,
              watch_id
      FROM pending_pullrequest_notification
      WHERE watch_id = $1
      AND pullrequest_history_id NOT IN (SELECT id FROM ship_notification)
      ORDER BY created_at`;
    const v = [watchId];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    const pending: Notification[] = [];
    for (const row of rows) {
      const result = this.mapRow(row);
      pending.push(result);
    }
    return pending;
  }

  @instrumented()
  async checkPendingPrNotification(ctx: jaeger.SpanContext, client: PostgresWrapper, owner: string, repo: string, prNumber: number ): Promise<QueryResult> {
    const q = `select
        pullrequest_history_id,
        branch,
        root_path,
        created_at,
        github_installation_id,
        watch_id
      from
        pending_pullrequest_notification
      where
        org = $1
        and repo = $2
        and pullrequest_number = $3`;
    const v = [owner, repo, prNumber];
    return client.query(q, v);
  }

  @instrumented()
  async createPrNotification(ctx: jaeger.SpanContext, client: PostgresWrapper, notificationID: string, owner:string, repo:string, branch: string, rootPath: string, createdAt: any, githubInstallationID: number): Promise<void> {
    const q = `insert into pullrequest_notification
      (notification_id, org, repo, branch, root_path, created_at, github_installation_id)
      values
      ($1, $2, $3, $4, $5, $6, $7)`;
    const v = [notificationID, owner, repo, branch, rootPath, createdAt, githubInstallationID];
    await client.query(q, v);
  }

  @instrumented()
  async createShipNotification(ctx: jaeger.SpanContext, client: PostgresWrapper, notificationID: string, watchID: string, createdAt: any): Promise<void> {
    const q = `insert into ship_notification
      (id, watch_id, created_at, enabled)
      values
      ($1, $2, $3, $4)`;
    const v = [notificationID, watchID, createdAt, 1];
    await client.query(q, v);
  }

  @instrumented()
  async updatePrNotification(ctx: jaeger.SpanContext, client: PostgresWrapper, notificationID: string, owner:string, repo:string, branch: string, rootPath: string, githubInstallationID: number): Promise<void> {
    const q = `UPDATE pullrequest_notification
    SET
      org = $2,
      repo = $3,
      branch = $4,
      root_path = $5,
      github_installation_id = $6
    WHERE notification_id = $1`;
    const v = [notificationID, owner, repo, branch, rootPath, githubInstallationID];
    await client.query(q, v);
  }

  @instrumented()
  async deletePendingPrNotification(ctx: jaeger.SpanContext, client: PostgresWrapper, owner: string, repo: string, prNumber: number ): Promise<QueryResult> {
    const q = `delete from pending_pullrequest_notification
      where
        org = $1
        and repo = $2
        and pullrequest_number = $3`;
    const v = [owner, repo, prNumber];
    return client.query(q, v);
  }

  private mapRow(row: any): Notification {
    const result: Notification = {
      id: row.id,
      createdOn: row.created_at,
      updatedOn: row.updated_at,
      triggeredOn: row.triggered_at,
      enabled: row.enabled,
      webhook: null,
      email: null,
      pullRequest: null,
      isDefault: false,
    };

    if (row.destination_uri) {
      result.webhook = {
        uri: row.destination_uri,
      };
      if (row.destination_uri === "placeholder") {
        result.isDefault = true;
      }
    }

    if (row.recipient) {
      result.email = {
        recipientAddress: row.recipient,
      };
      if (row.recipient === "placeholder") {
        result.isDefault = true;
      }
    }

    if (row.org) {
      result.pullRequest = {
        org: row.org,
        repo: row.repo,
        branch: row.branch,
        rootPath: row.root_path,
      };
      if (row.org === "placeholder") {
        result.isDefault = true;
      }
      result.pending = !!row.pullrequest_number
    }

    return result;
  }

  private mapPullRequestHistoryItem(row: any): PullRequestHistoryItem {
    return {
      title: row.version_label !== "" && row.version_label !== "Unknown" ? `Version ${row.version_label}` : "",
      createdOn: row.created_at,
      number: row.pullrequest_number,
      uri: `https://github.com/${row.org}/${row.repo}/pull/${row.pullrequest_number}`,
      status: row.github_status,
      sequence: row.sequence,
      sourceBranch: row.source_branch,
    };
  }
}
