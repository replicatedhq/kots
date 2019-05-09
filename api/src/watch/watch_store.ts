import { S3 } from "aws-sdk";
import { stripIndent } from "common-tags";
import * as jaeger from "jaeger-client";
import * as _ from "lodash";
import * as path from "path";
import { instrumented } from "monkit";
import * as randomstring from "randomstring";
import slugify from "slugify";
import { ContributorItem, StateMetadata, WatchItem, VersionItem } from "../generated/types";
import { ReplicatedError } from "../server/errors";
import { Params } from "../server/params";
import * as pg from "pg";
import { checkExists, putObject } from "../util/s3";
import { tracer } from "../server/tracing";

export interface FindWatchOpts {
  id?: string;
  slug?: string;
}

export interface GeneratedFile {
  watchId: string;
  createdAt: string;
  sequence: number;
  filepath: string;
}

export class WatchStore {
  constructor(
    private readonly pool: pg.Pool,
    private readonly params: Params
  ) {}

  async setCurrentVersion(ctx: jaeger.SpanContext, watchId: string, sequence: number): Promise<void> {
    const q = `update watch set current_sequence = $1 where id = $2`;
    const v = [
      sequence,
      watchId,
    ];

    await this.pool.query(q, v);
  }

  async updateVersionStatus(ctx: jaeger.SpanContext, watchId: string, sequence: number, status: string): Promise<void> {
    const q = `update watch_version set status = $1 where watch_id = $2 and sequence = $3`;
    const v = [
      status,
      watchId,
      sequence,
    ];

    await this.pool.query(q, v);
  }

  async getOneVersion(ctx: jaeger.SpanContext, watchId: string, sequence: number): Promise<VersionItem> {
    const q = `select created_at, version_label, status, sequence, pullrequest_number from watch_version where watch_id = $1 and sequence = $2`;
    const v = [
      watchId,
      sequence,
    ];

    const result = await this.pool.query(q, v);
    const versionItem = this.mapWatchVersion(result.rows[0]);
    return versionItem;
  }

  async getCurrentVersion(watchId: string): Promise<VersionItem|undefined> {
    let q = `select current_sequence from watch where id = $1`;
    let v = [
      watchId,
    ];

    let result = await this.pool.query(q, v);
    const sequence = result.rows[0].current_sequence;

    if (sequence === null) {
      return;
    }

    q = `select created_at, version_label, status, sequence, pullrequest_number from watch_version where watch_id = $1 and sequence = $2`;
    v = [
      watchId,
      sequence,
    ];

    result = await this.pool.query(q, v);
    const versionItem = this.mapWatchVersion(result.rows[0]);

    return versionItem;
  }

  async listPastVersions(watchId: string): Promise<VersionItem[]> {
    let q = `select current_sequence from watch where id = $1`;
    let v = [
      watchId,
    ];

    let result = await this.pool.query(q, v);
    const sequence = result.rows[0].current_sequence;

    // If there is not a current_sequence, then there can't be past versions
    if (sequence === null) {
      return [];
    }

    q = `select created_at, version_label, status, sequence, pullrequest_number from watch_version where watch_id = $1 and sequence < $2 order by sequence desc`;
    v = [
      watchId,
      sequence,
    ];

    const { rows }: { rows: any[] } = await this.pool.query(q, v);
    const versionItems: VersionItem[] = [];

    for (const row of rows) {
      versionItems.push(this.mapWatchVersion(row));
    }

    return versionItems;
  }

  async listPendingVersions(ctx: jaeger.SpanContext, watchId: string): Promise<VersionItem[]> {
    let q = `select current_sequence from watch where id = $1`;
    let v = [
      watchId,
    ];

    let result = await this.pool.query(q, v);
    let sequence = result.rows[0].current_sequence;

    // If there is not a current_sequence, then all versions are future versions
    if (sequence === null) {
      sequence = -1;
    }

    q = `select created_at, version_label, status, sequence, pullrequest_number from watch_version where watch_id = $1 and sequence > $2 order by sequence desc`;
    v = [
      watchId,
      sequence,
    ];

    const { rows }: { rows: any[] } = await this.pool.query(q, v);
    const versionItems: VersionItem[] = [];

    for (const row of rows) {
      versionItems.push(this.mapWatchVersion(row));
    }

    return versionItems;
  }

  async createWatchVersion(ctx: jaeger.SpanContext, watchId: string, createdAt: any, versionLabel: string, status: string, sourceBranch: string, sequence: number, pullRequestNumber: number): Promise<VersionItem | void> {
    const q = `insert into watch_version (watch_id, created_at, version_label, status, source_branch, sequence, pullrequest_number) values ($1, $2, $3, $4, $5, $6, $7)`;
    const v = [
      watchId,
      createdAt,
      versionLabel,
      status,
      sourceBranch,
      sequence,
      pullRequestNumber,
    ];

    await this.pool.query(q, v);
  }

  async setParent(ctx: jaeger.SpanContext, watchId: string, parentId?: string): Promise<void> {
    const pg = await this.pool.connect();

    try {
      const q = `update watch set parent_watch_id = $1 where id = $2`;
      const v = [
        parentId,
        watchId,
      ];

      await pg.query(q, v);;
    } finally {
      pg.release();
    }
  }

  async setCluster(ctx: jaeger.SpanContext, watchId: string, clusterId: string, githubPath?: string): Promise<void> {
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      try {
        let q = `delete from watch_cluster where watch_id = $1 and cluster_id = $2`;
        let v: any[] = [
          watchId,
          clusterId,
        ];
        await pg.query(q, v);

        q = `insert into watch_cluster (watch_id, cluster_id, github_path) values ($1, $2, $3)`;
        v = [
          watchId,
          clusterId,
          githubPath,
        ];
        await pg.query(q, v);

        await pg.query("commit");
      } catch (err) {
        await pg.query("rollback");
        throw err;
      }
    } finally {
      pg.release();
    }
  }

  async createDownstreamToken(ctx: jaeger.SpanContext, watchId: string): Promise<string> {
    const token = randomstring.generate({ capitalization: "lowercase" });
    const pg = await this.pool.connect();

    try {
      const q = `insert into watch_downstream_token (watch_id, token) values ($1, $2)`;
      const v = [
        watchId,
        token,
      ];

      await pg.query(q, v);

      return token;
    } finally {
      pg.release();
    }
  }

  async listForCluster(ctx: jaeger.SpanContext, clusterId: string): Promise<WatchItem[]> {
    const pg = await this.pool.connect();

    try {
      const q = `select watch_id from watch_cluster where cluster_id = $1`;
      const v = [
        clusterId,
      ];

      const { rows }: { rows: any[] } = await pg.query(q, v);
      const watchIds: string[] = [];
      for (const row of rows) {
        watchIds.push(row.watch_id);
      }

      const watches: WatchItem[] = [];
      for (const watchId of watchIds) {
        const watch = await this.getWatch(null, watchId);
        watches.push(watch);
      }

      return watches;
    } finally {
      pg.release();
    }
  }

  async findUpstreamWatch(ctx: jaeger.SpanContext, token: string, watchId: string): Promise<WatchItem> {
    const pg = await this.pool.connect();

    try {
      const q = `select watch_id from watch_downstream_token where token = $1`;
      const v = [token];

      const { rows }: { rows: any[] } = await pg.query(q, v);
      if (rows.length === 0) {
        throw new ReplicatedError("Watch not found");
      }

      // This next check may not be necessary?
      if (watchId !== rows[0].watch_id) {
        throw new ReplicatedError("Watch not found");
      }

      const watch = await this.getWatch(null, rows[0].watch_id);

      return watch;
    } finally {
      pg.release();
    }
  }

  async findUserWatch(ctx: jaeger.SpanContext, userId: string, opts: FindWatchOpts): Promise<WatchItem> {
    if (!opts.id && !opts.slug) {
      throw new TypeError("one of slug or id is required");
    }

    const pg = await this.pool.connect();

    try {
      let q;
      let v;

      if (opts.id) {
        q = "SELECT watch_id FROM user_watch WHERE watch_id = $1 and user_id = $2";
        v = [opts.id, userId];
      } else if (opts.slug) {
        q = "SELECT watch_id FROM user_watch INNER JOIN watch ON watch.id = user_watch.watch_id WHERE watch.slug = $1 and user_watch.user_id = $2";
        v = [opts.slug, userId];
      }

      const { rows }: { rows: any[] } = await pg.query(q, v);
      if (rows.length === 0) {
        throw new ReplicatedError("Watch not found");
      }

      const watch = await this.getWatch(null, rows[0].watch_id);
      return watch;
    } finally {
      pg.release();
    }
  }

  async getWatch(ctx: jaeger.SpanContext, id: string): Promise<WatchItem> {
    const pg = await this.pool.connect();

    try {
      const q = "select id, current_state, title, icon_uri, slug, created_at, updated_at from watch where id = $1";
      const v = [id];

      const { rows }: { rows: any[] } = await pg.query(q, v);
      const watches = rows.map(row => this.mapWatch(row));
      const watch = watches[0];

      watch.watches = await this.listWatches(null, undefined, watch.id!);

      return watch;
    } finally {
      pg.release();
    }
  }

  @instrumented()
  async deleteWatch(ctx: jaeger.SpanContext, watchId: string): Promise<boolean> {
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      try {
        let q = "delete from watch WHERE id = $1";
        const v = [watchId];
        await pg.query(q, v);

        q = "delete from user_watch where watch_id = $1";
        await pg.query(q, v);

        q = "delete from watch_version where watch_id = $1";
        await pg.query(q, v);

        q = "delete from watch_cluster where watch_id = $1";
        await pg.query(q, v);

        q = "delete from watch_downstream_token where watch_id = $1";
        await pg.query(q, v);

        await pg.query("commit");
      } catch {
        await pg.query("rollback");
      }

      return true;
    } finally {
      pg.release();
    }
  }

  async listAllWatchesForAllTeams(ctx: jaeger.SpanContext): Promise<WatchItem[]> {
    const pg = await this.pool.connect();

    try {
      const q = `select id, current_state, title, slug, icon_uri, created_at, updated_at from watch`;
      const v = [];

      const { rows }: { rows: any[] } = await pg.query(q, v);
      const watches: WatchItem[] = [];
      for (const row of rows) {
        const result = this.mapWatch(row);
        watches.push(result);
      }

      return watches;
    } finally {
      pg.release();
    }
  }

  async listUsersForWatch(ctx: jaeger.SpanContext, watchId: string): Promise<string[]> {
    const pg = await this.pool.connect();

    try {
      const q = `select user_id from user_watch where watch_id = $1`;
      const v = [watchId];

      const { rows }: { rows: any[] } = await pg.query(q, v);
      const userIds: string[] = [];
      for (const row of rows) {
        userIds.push(row.user_id);
      }

      return userIds;
    } finally {
      pg.release();
    }
  }

  async listAllUserWatches(ctx: jaeger.SpanContext, userId?: string): Promise<WatchItem[]> {
    const pg = await this.pool.connect();

    try {
      const q = `
          SELECT user_id,
                watch_id as id,
                watch.current_state,
                watch.title,
                watch.slug,
                watch.icon_uri,
                watch.created_at,
                watch.updated_at
          FROM user_watch
                JOIN watch ON watch.id = user_watch.watch_id
          WHERE user_watch.user_id = $1
          ORDER BY watch.title
        `;
      const v = [
        userId,
      ];

      const { rows }: { rows: any[] } = await pg.query(q, v);
      const watches: WatchItem[] = [];
      for (const row of rows) {
        const watch = this.mapWatch(row);

        watch.watches = await this.listWatches(null, userId, watch.id!);
        watches.push(watch);
      }

      return watches;
    } finally {
      pg.release();
    }
  }

  async listWatches(ctx: jaeger.SpanContext, userId?: string, parentId?: string): Promise<WatchItem[]> {
    const pg = await this.pool.connect();

    try {
        let q;
        let v;

        if (parentId) {
          q = `
            SELECT user_id,
                  watch_id as id,
                  watch.current_state,
                  watch.title,
                  watch.slug,
                  watch.icon_uri,
                  watch.created_at,
                  watch.updated_at
            FROM user_watch
                  JOIN watch ON watch.id = user_watch.watch_id
            AND watch.parent_watch_id = $1
            ORDER BY watch.title
          `;
          v = [
            parentId,
          ];
        } else {
          q = `
            SELECT user_id,
                  watch_id as id,
                  watch.current_state,
                  watch.title,
                  watch.slug,
                  watch.icon_uri,
                  watch.created_at,
                  watch.updated_at
            FROM user_watch
                  JOIN watch ON watch.id = user_watch.watch_id
            WHERE user_watch.user_id = $1
            AND watch.parent_watch_id IS NULL
            ORDER BY watch.title
          `;
          v = [
            userId,
          ];
        }

        const { rows }: { rows: any[] } = await pg.query(q, v);
        const watches: WatchItem[] = [];
        for (const row of rows) {
          const watch = this.mapWatch(row);

          watch.watches = await this.listWatches(null, userId, watch.id!);
          watches.push(watch);
        }

        return watches;
      } finally {
        pg.release();
      }
  }

  // returns the list of generated files for this watch in reverse sequence order. (highest sequence number first)
  async listGeneratedFiles(ctx: jaeger.SpanContext, watchId: string): Promise<GeneratedFile[]> {
    const pg = await this.pool.connect();

    try {
      const q = stripIndent`
        SELECT ship_output_files.watch_id as watch_id,
              ship_output_files.created_at as created_at,
              ship_output_files.sequence as sequence,
              ship_output_files.filepath as filepath
        FROM ship_output_files
              JOIN user_watch ON user_watch.watch_id = ship_output_files.watch_id
        WHERE ship_output_files.watch_id = $1
        ORDER BY sequence DESC`;

      const v = [watchId];
      const { rows }: { rows: any[] } = await pg.query(q, v);
      const files: GeneratedFile[] = [];
      for (const row of rows) {
        const result = this.mapGeneratedFile(row);
        files.push(result);
      }

      return files;
    } finally {
      pg.release();
    }
  }

  async getLatestGeneratedFileS3Params(ctx: jaeger.SpanContext, watchId: string, sequence?: number): Promise<S3.GetObjectRequest> {
    let generatedFiles: GeneratedFile[];
    if (_.isUndefined(sequence)) {
      generatedFiles = await this.listGeneratedFiles(null, watchId);
    } else {
      generatedFiles = [await this.getGeneratedFileForSequence(null, watchId, sequence)];
    }

    let exists = false;
    let params: S3.GetObjectRequest | undefined;
    for (const file of generatedFiles) {
      const { filepath } = file;

      if (this.params.objectStoreInDatabase) {
         // used in testing only, not recommended for any real use
         const q = `select encoded_block from object_store where filepath = $1`;
         const v = [
           filepath,
         ];

         const pg = await this.pool.connect();

         try {
          const result = await pg.query(q, v);
          const buffer = new Buffer(result.rows[0].encoded_block, "base64");

          // Write to the local s3 server so we can return an S3.GetObjectRequest
          const rewrittenFilepath = path.join(this.params.shipOutputBucket.trim(), filepath);
          await putObject(this.params, rewrittenFilepath, buffer, "ship-pacts");

          params = {
            Bucket: this.params.shipOutputBucket.trim(),
            Key: rewrittenFilepath,
          };

          return params;
        } finally {
          pg.release();
        }
      } else {
        params = {
          Bucket: this.params.shipOutputBucket.trim(),
          Key: filepath,
        };

        exists = await checkExists(this.params, params);
        if (exists) {
          break;
        }
      }
    }

    if (!exists || !params) {
      throw new ReplicatedError("File not found", "404");
    }
    return params;
  }

  @instrumented()
  async getGeneratedFileForSequence(ctx: jaeger.SpanContext, watchId: string, sequence: number): Promise<GeneratedFile> {
    const pg = await this.pool.connect();

    try {
      const q = stripIndent`
        SELECT watch_id, created_at, sequence, filepath
        FROM ship_output_files
        WHERE watch_id = $1
          AND sequence = $2`;

      const v = [
        watchId,
        sequence
      ];
      const { rows }: { rows: any[] } = await pg.query(q, v);
      const result = rows.map(row => this.mapGeneratedFile(row));

      return result[0];
    } finally {
      pg.release();
    }
  }

  async searchWatches(ctx: jaeger.SpanContext, userId: string, watchName: string): Promise<WatchItem[]> {
    const pg = await this.pool.connect();

    try {
      const q = `
        SELECT user_id,
              watch_id as id,
              watch.current_state,
              watch.title,
              watch.slug,
              watch.icon_uri,
              watch.created_at,
              watch.updated_at
        FROM user_watch
              JOIN watch ON watch.id = user_watch.watch_id
        WHERE user_watch.user_id = $1
          AND watch.title ILIKE $2`;

      const v = [
        userId,
        `%${watchName}%`,
      ];

      const { rows }: { rows: any[] } = await pg.query(q, v);
      const watches: WatchItem[] = [];
      for (const row of rows) {
        const result = this.mapWatch(row);
        watches.push(result);
      }
      return watches;
    } finally {
      pg.release();
    }
  }

  async getStateJSON(ctx: jaeger.SpanContext, id: string): Promise<any> {
    const pg = await this.pool.connect();

    try {
      const q = "SELECT current_state FROM watch WHERE id = $1";
      const v = [id];

      const { rows }: { rows: any[] } = await pg.query(q, v);
      return JSON.parse(rows[0].current_state);
    } finally {
      pg.release();
    }
  }

  async updateStateJSON(ctx: jaeger.SpanContext, id: string, stateJSON: string, metadata: StateMetadata) {
    const pg = await this.pool.connect();

    try {
      const title = metadata.name;

      const q = "UPDATE watch SET current_state = $1, updated_at = $2, title = $3, icon_uri = $4 WHERE id = $5";
      const v = [stateJSON, new Date(), title, metadata.icon, id];

      await pg.query(q, v);
    } finally {
      pg.release();
    }
  }

  async updateWatch(ctx: jaeger.SpanContext, id: string, watchName?: string, iconUri?: string) {
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      if (watchName) {
        const q = "UPDATE watch SET title = $2 WHERE id = $1";
        const v = [id, watchName];
        await pg.query(q, v);
      }

      if (iconUri) {
        const q = "UPDATE watch SET icon_uri = $2 WHERE id = $1";
        const v = [id, iconUri];
        await pg.query(q, v);
      }

      await pg.query("commit");
    } finally {
      await pg.query("rollback");
      pg.release();
    }
  }

  async createNewWatch(ctx: jaeger.SpanContext, stateJSON: string, owner: string, userId: string, metadata: StateMetadata): Promise<WatchItem> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    const title = _.get(metadata, "name", "New Application");
    const icon = _.get(metadata, "icon", "https://vignette.wikia.nocookie.net/jet/images/e/ea/Under_construction-icon.JPG/revision/latest?cb=20100622032326"); // under construction image
    const titleForSlug = title.replace(/\./g, "-");

    const slugProposal = `${owner.toLowerCase()}/${slugify(titleForSlug, { lower: true })}`;
    const watches = await this.listAllUserWatches(ctx, userId);
    const existingSlugs = watches.map(watch => watch.slug);
    let finalSlug = slugProposal;

    if (_.includes(existingSlugs, slugProposal)) {
      const maxNumber =
        _(existingSlugs)
          .map(slug => {
            const result = slug!.replace(slugProposal, "").replace("-", "");

            return result ? parseInt(result, 10) : 0;
          })
          .max() || 0;

      finalSlug = `${slugProposal}-${maxNumber + 1}`;
    }

    const pg = await this.pool.connect();

    try {
      await pg.query("begin");
      const q = "INSERT INTO watch (id, current_state, title, slug, icon_uri, created_at) VALUES ($1, $2, $3, $4, $5, $6)";
      const v = [id, stateJSON, title, finalSlug, icon, new Date()];

      await pg.query(q, v);

      const uwq = "INSERT INTO user_watch (user_id, watch_id) VALUES ($1, $2)";
      const uwv = [userId, id];
      await pg.query(uwq, uwv);

      await pg.query("commit");
      const watch = await this.getWatch(null, id);

      return watch;
    } finally {
      await pg.query("rollback");
      pg.release();
    }
  }

  async listWatchContributors(id: string): Promise<ContributorItem[]> {
    const q = `
      SELECT ship_user.id as user_id, ship_user.created_at, github_user.github_id, github_user.username, github_user.avatar_url
      FROM user_watch
            JOIN ship_user ON ship_user.id = user_watch.user_id
            JOIN github_user ON github_user.user_id = ship_user.id
      WHERE watch_id = $1
    `;
    const v = [id];

    const { rows }: { rows: any[] } = await this.pool.query(q, v);

    const contributors: ContributorItem[] = [];
    for (const row of rows) {
      const result = this.mapContributor(row);
      contributors.push(result);
    }
    return contributors;
  }

  private mapWatch(row: any): WatchItem {
    const parsedWatchName = this.parseWatchName(row.title);

    return {
      id: row.id,
      stateJSON: row.current_state,
      watchName: parsedWatchName,
      slug: row.slug,
      watchIcon: row.icon_uri,
      lastUpdated: row.updated_at,
      createdOn: row.created_at,
    };
  }

  private mapGeneratedFile(row: any): GeneratedFile {
    return {
      watchId: row.watch_id,
      createdAt: row.created_at,
      sequence: row.sequence,
      filepath: row.filepath,
    };
  }

  private mapContributor(row: any): ContributorItem {
    return {
      id: row.user_id,
      createdAt: row.created_at,
      githubId: row.github_id,
      login: row.username,
      avatar_url: row.avatar_url,
    };
  }

  private mapWatchVersion(row: any): VersionItem {
    return {
      title: row.version_label,
      status: row.status,
      createdOn: row.created_at,
      sequence: row.sequence,
      pullrequestNumber: row.pullrequest_number,
    };
  }

  private parseWatchName(watchName: string): string {
    if (watchName.startsWith("replicated.app") || watchName.startsWith("staging.replicated.app") || watchName.startsWith("local.replicated.app")) {
      const splitReplicatedApp = watchName.split("/");
      if (splitReplicatedApp.length < 2) {
        return watchName;
      }

      const splitReplicatedAppParams = splitReplicatedApp[1].split("?");
      return splitReplicatedAppParams[0];
    }

    return watchName;
  }
}
