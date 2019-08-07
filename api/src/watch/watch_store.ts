import { S3 } from "aws-sdk";
import { stripIndent } from "common-tags";
import _ from "lodash";
import path from "path";
import randomstring from "randomstring";
import slugify from "slugify";
import { Watch, Version, StateMetadata, Contributor, parseWatchName } from "./"
import { ReplicatedError } from "../server/errors";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import pg from "pg";
import { checkExists, putObject } from "../util/s3";

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
  ) { }

  async queueCheckForUpdates(watchId: string): Promise<void> {
    const now = new Date();
    const q = `update watch set last_watch_check_at = $1 where id = $2`;
    const v = [
      new Date(now.setHours(now.getHours() - 1)),
      watchId,
    ];

    await this.pool.query(q, v);
  }
  async setCurrentVersion(watchId: string, sequence: number, deployedAt?: string, status?: string): Promise<void> {
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      try {
        let q = `update watch set current_sequence = $1 where id = $2`;
        let v: any[] = [
          sequence,
          watchId,
        ];
        await pg.query(q, v);

        q = `update watch_version set deployed_at = $1${status ? ", status = $4" : " "}where watch_id = $2 and sequence = $3`;
        v = [
          deployedAt || new Date(),
          watchId,
          sequence,
        ];
        if (status) {
          v.push(status);
        }
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

  async updateVersionStatus(watchId: string, sequence: number, status: string): Promise<void> {
    const q = `update watch_version set status = $1, last_synced_at = current_timestamp where watch_id = $2 and sequence = $3`;
    const v = [
      status,
      watchId,
      sequence,
    ];

    await this.pool.query(q, v);
  }

  async getOneVersion(watchId: string, sequence: number): Promise<Version> {
    const q = `select created_at, version_label, status, sequence, pullrequest_number, deployed_at from watch_version where watch_id = $1 and sequence = $2`;
    const v = [
      watchId,
      sequence,
    ];

    const result = await this.pool.query(q, v);
    const versionItem = this.mapWatchVersion(result.rows[0]);
    return versionItem;
  }

  async getCurrentVersion(watchId: string): Promise<Version | undefined> {
    let q = `select current_sequence from watch where id = $1`;
    let v = [
      watchId,
    ];
    let result = await this.pool.query(q, v);
    const sequence = result.rows[0].current_sequence;

    if (sequence === null) {
      return;
    }

    q = `select created_at, version_label, status, sequence, pullrequest_number, deployed_at from watch_version where watch_id = $1 and sequence = $2`;
    v = [
      watchId,
      sequence,
    ];

    result = await this.pool.query(q, v);
    const versionItem = this.mapWatchVersion(result.rows[0]);

    return versionItem;
  }

  async listPastVersions(watchId: string): Promise<Version[]> {
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

    q = `select created_at, version_label, status, sequence, pullrequest_number, deployed_at from watch_version where watch_id = $1 and sequence < $2 order by sequence desc`;
    v = [
      watchId,
      sequence,
    ];

    result = await this.pool.query(q, v);
    const versionItems: Version[] = [];

    for (const row of result.rows) {
      versionItems.push(this.mapWatchVersion(row));
    }

    return versionItems;
  }

  async getVersionForCommit(watchId: string, commitSha: string): Promise<Version|undefined> {
    const q = `select created_at, version_label, status, sequence, pullrequest_number, deployed_at from watch_version where watch_id = $1 and commit_sha = $2`;
    const v = [
      watchId,
      commitSha,
    ];

    const result = await this.pool.query(q, v);

    if (result.rowCount === 0) {
      return;
    }

    return this.mapWatchVersion(result.rows[0]);
  }

  async listPendingVersions(watchId: string): Promise<Version[]> {
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

    q = `
select created_at, version_label, status, sequence, pullrequest_number, deployed_at
from watch_version
where watch_id = $1 and sequence > $2
order by sequence desc`;
    v = [
      watchId,
      sequence,
    ];

    result = await this.pool.query(q, v);
    const versionItems: Version[] = [];

    for (const row of result.rows) {
      versionItems.push(this.mapWatchVersion(row));
    }

    return versionItems;
  }

  async createWatchVersion(watchId: string, createdAt: any, versionLabel: string, status: string, sourceBranch: string, sequence: number, pullRequestNumber: number): Promise<Version | void> {
    const q = `insert into watch_version (watch_id, created_at, version_label, status, source_branch, sequence, pullrequest_number, last_synced_at) values ($1, $2, $3, $4, $5, $6, $7, current_timestamp)`;
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

  async setParent(watchId: string, parentId?: string): Promise<void> {
    const q = `update watch set parent_watch_id = $1 where id = $2`;
    const v = [
      parentId,
      watchId,
    ];

    await this.pool.query(q, v);
  }

  async setCluster(watchId: string, clusterId: string, githubPath?: string): Promise<void> {
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

  async createDownstreamToken(watchId: string): Promise<string> {
    const token = randomstring.generate({ capitalization: "lowercase" });
    const q = `insert into watch_downstream_token (watch_id, token) values ($1, $2)`;
    const v = [
      watchId,
      token,
    ];

    await this.pool.query(q, v);

    return token;
  }

  async listForCluster(clusterId: string): Promise<Watch[]> {
    const q = `select watch_id from watch_cluster where cluster_id = $1`;
    const v = [
      clusterId,
    ];

    const result = await this.pool.query(q, v);
    const watchIds: string[] = [];
    for (const row of result.rows) {
      watchIds.push(row.watch_id);
    }

    const watches: Watch[] = [];
    for (const watchId of watchIds) {
      const watch = await this.getWatch(watchId);
      watches.push(watch);
    }

    return watches;
  }

  async findUpstreamWatch(token: string, watchId: string): Promise<Watch> {
    const q = `select watch_id from watch_downstream_token where token = $1`;
    const v = [token];

    const result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError("Watch not found");
    }

    // This next check may not be necessary?
    if (watchId !== result.rows[0].watch_id) {
      throw new ReplicatedError("Watch not found");
    }

    const watch = await this.getWatch(result.rows[0].watch_id);

    return watch;
  }

  async findUserWatch(userId: string, opts: FindWatchOpts): Promise<Watch> {
    if (!opts.id && !opts.slug) {
      throw new TypeError("one of slug or id is required");
    }

    let watchId: string = "";

    if (opts.id) {
      const q = "select watch_id from user_watch where watch_id = $1 and user_id = $2";
      const v = [opts.id, userId];
      const result = await this.pool.query(q, v);
      if (result.rows.length === 1) {
        watchId = result.rows[0].watch_id;
      }

      if (watchId === "") {
        const qq = "select watch.id as watch_id from watch inner join user_watch on watch.parent_watch_id = user_watch.watch_id where watch.id = $1 and user_watch.user_id = $2";
        const vv = [opts.id, userId];
        const result = await this.pool.query(qq, vv);
        if (result.rows.length === 1) {
          watchId = result.rows[0].watch_id;
        }
      }
    } else if (opts.slug) {
      const q = "select watch_id from user_watch inner join watch on watch.id = user_watch.watch_id where watch.slug = $1 and user_watch.user_id = $2";
      const v = [opts.slug, userId];
      const result = await this.pool.query(q, v);
      if (result.rows.length === 1) {
        watchId = result.rows[0].watch_id;
      }
    }

    if (watchId === "") {
      throw new ReplicatedError("Watch not found");
    }

    const watch = await this.getWatch(watchId);
    return watch;
  }

  async getWatch(id: string): Promise<Watch> {
    const watchQuery = `
      SELECT
        id,
        current_state,
        title,
        icon_uri,
        slug,
        created_at,
        updated_at,
        metadata,
        last_watch_check_at FROM watch WHERE id = $1`;

    const watchQueryResults = await this.pool.query(watchQuery, [id]);

    if (watchQueryResults.rowCount === 0) {
      throw new ReplicatedError(`Couldn't find watch with watch_id of ${id}`);
    }
    // Get preflight spec for this watch
    const preflightQuery =
      `SELECT COUNT(1) FROM preflight_spec WHERE watch_id = $1`;

    const preflightResults = await this.pool.query(preflightQuery, [ id ]);

    const watch = new Watch();
    watch.id = watchQueryResults.rows[0].id;
    watch.stateJSON = watchQueryResults.rows[0].current_state;
    watch.watchName = parseWatchName(watchQueryResults.rows[0].title)
    watch.slug = watchQueryResults.rows[0].slug;
    watch.watchIcon = watchQueryResults.rows[0].icon_uri;
    watch.lastUpdated = watchQueryResults.rows[0].updated_at;
    watch.createdOn = watchQueryResults.rows[0].created_at;
    watch.metadata = watchQueryResults.rows[0].metadata;
    watch.lastUpdateCheck = watchQueryResults.rows[0].last_watch_check_at;
    // pgQueryResult.count returns a string by design
    watch.hasPreflight = preflightResults.rows[0].count !== "0";

    return watch;
  }

  async getIdFromSlug(slug: string): Promise<string> {
    const q = "select id from watch where slug = $1";
    const v = [slug];

    const result = await this.pool.query(q, v);
    return result.rows[0].id;
  }

  async getParentWatchId(id: string): Promise<string> {
    const q = "select parent_watch_id from watch where id = $1";
    const v = [id];

    const result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError("This watch does not have a parent");
    }
    return result.rows[0].parent_watch_id;
  }

  async deleteWatch(watchId: string): Promise<boolean> {
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

  async listUsersForWatch(watchId: string): Promise<string[]> {
    const q = `select user_id from user_watch where watch_id = $1`;
    const v = [watchId];

    const result = await this.pool.query(q, v);
    const userIds: string[] = [];
    for (const row of result.rows) {
      userIds.push(row.user_id);
    }

    return userIds;
  }

  async listAllUserWatches(userId: string): Promise<Watch[]> {
    const q = `select watch_id as id from user_watch where user_id = $1`;
    const v = [
      userId,
    ];

    const result = await this.pool.query(q, v);
    const watches: Watch[] = [];
    for (const row of result.rows) {
      const watch = await this.getWatch(row.id);
      watches.push(watch);
    }

    return _.sortBy(watches, ["title"]);
  }

  async listWatches(userId?: string, parentId?: string): Promise<Watch[]> {
    let q;
    let v;

    if (parentId) {
      q = `select id from watch where parent_watch_id = $1`;
      v = [
        parentId,
      ];
    } else {
      q = `select watch_id as id from user_watch join watch on watch.id = user_watch.watch_id
        where user_watch.user_id = $1
        and watch.parent_watch_id is null`;
      v = [
        userId,
      ];
    }

    const result = await this.pool.query(q, v);
    const watches: Watch[] = [];
    for (const row of result.rows) {
      const watch = await this.getWatch(row.id);
      watches.push(watch);
    }

    return _.sortBy(watches, ["title"]);
  }

  // returns the list of generated files for this watch in reverse sequence order. (highest sequence number first)
  async listGeneratedFiles(watchId: string): Promise<GeneratedFile[]> {
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
    const result = await this.pool.query(q, v);
    const files: GeneratedFile[] = [];
    for (const row of result.rows) {
      const result = this.mapGeneratedFile(row);
      files.push(result);
    }

    return files;
  }

  async getLatestGeneratedFileS3Params(watchId: string, sequence?: number): Promise<S3.GetObjectRequest> {
    let generatedFiles: GeneratedFile[];
    if (_.isUndefined(sequence)) {
      generatedFiles = await this.listGeneratedFiles(watchId);
    } else {
      generatedFiles = [await this.getGeneratedFileForSequence(watchId, sequence!)];
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

        const result = await this.pool.query(q, v);
        const buffer = new Buffer(result.rows[0].encoded_block, "base64");

        // Write to the local s3 server so we can return an S3.GetObjectRequest
        const rewrittenFilepath = path.join(this.params.shipOutputBucket.trim(), filepath);
        await putObject(this.params, rewrittenFilepath, buffer, "ship-pacts");

        params = {
          Bucket: this.params.shipOutputBucket.trim(),
          Key: rewrittenFilepath,
        };

        return params;
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
      throw new ReplicatedError("File not found");
    }
    return params;
  }

  async getGeneratedFileForSequence(watchId: string, sequence: number): Promise<GeneratedFile> {
    const q = stripIndent`
      SELECT watch_id, created_at, sequence, filepath
      FROM ship_output_files
      WHERE watch_id = $1
        AND sequence = $2`;

    const v = [
      watchId,
      sequence
    ];
    const result = await this.pool.query(q, v);
    const generatedFile = result.rows.map(row => this.mapGeneratedFile(row));

    return generatedFile[0];
  }

  async searchWatches(userId: string, watchName: string): Promise<Watch[]> {
    const q = `
      SELECT user_id,
            watch_id as id,
            watch.current_state,
            watch.title,
            watch.slug,
            watch.icon_uri,
            watch.created_at,
            watch.updated_at,
            watch.metadata
      FROM user_watch
            JOIN watch ON watch.id = user_watch.watch_id
      WHERE user_watch.user_id = $1
        AND watch.title ILIKE $2`;

    const v = [
      userId,
      `%${watchName}%`,
    ];

    const result = await this.pool.query(q, v);
    const watches: Watch[] = [];
    for (const row of result.rows) {
      const hasPreflightQuery = `SELECT COUNT(1) FROM preflight_spec WHERE watch_id = $1`;
      const hasPreflightResult = await this.pool.query(hasPreflightQuery, [ row.id ]);

      const parsedWatchName = parseWatchName(row.title);
      const watch = new Watch();
      watch.id = row.id;
      watch.stateJSON = row.current_state;
      watch.watchName = parsedWatchName;
      watch.slug = row.slug;
      watch.watchIcon = row.icon_uri;
      watch.lastUpdated = row.updated_at;
      watch.createdOn = row.created_at;
      watch.metadata = row.metadata;
      watch.lastUpdateCheck = row.last_watch_check_at;
      // pgQueryResult.count returns a string by design
      watch.hasPreflight = hasPreflightResult.rows[0].count !== "0";

      watches.push(watch);
    }
    return watches;
  }

  async getStateJSON(id: string): Promise<any> {
    const q = "SELECT current_state FROM watch WHERE id = $1";
    const v = [id];

    const result = await this.pool.query(q, v);
    return JSON.parse(result.rows[0].current_state);
  }

  async updateStateJSON(id: string, stateJSON: string, metadata: StateMetadata) {
    let title = metadata.name;
    if (!title) {
      title = "Unknown application";
    }

    const q = "UPDATE watch SET current_state = $1, updated_at = $2, title = $3, icon_uri = $4 WHERE id = $5";
    const v = [stateJSON, new Date(), title, metadata.icon, id];

    await this.pool.query(q, v);
  }

  async updateWatch(id: string, watchName?: string, iconUri?: string) {
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

  async createNewWatch(stateJSON: string, owner: string, userId: string, metadata: StateMetadata): Promise<Watch> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    const title = _.get(metadata, "name", "New Application");
    const icon = _.get(metadata, "icon", "https://vignette.wikia.nocookie.net/jet/images/e/ea/Under_construction-icon.JPG/revision/latest?cb=20100622032326"); // under construction image
    const titleForSlug = title.replace(/\./g, "-");

    let slugProposal = `${owner.toLowerCase()}/${slugify(titleForSlug, { lower: true })}`;

    let i = 0;
    let foundUniqueSlug = false;
    while (!foundUniqueSlug) {
      if (i > 0) {
        slugProposal = `${owner.toLowerCase()}/${slugify(titleForSlug, { lower: true })}-${i}`;
      }
      const qq = `select count(1) as count from watch where slug = $1`;
      const vv = [
        slugProposal,
      ];

      const rr = await this.pool.query(qq, vv);
      if (parseInt(rr.rows[0].count) === 0) {
        foundUniqueSlug = true;
      }
      i++;
    }

    const pg = await this.pool.connect();

    try {
      await pg.query("begin");
      const q = "insert into watch (id, current_state, title, slug, icon_uri, created_at) values ($1, $2, $3, $4, $5, $6)";
      const v = [
        id,
        stateJSON,
        title,
        slugProposal,
        icon,
        new Date()
      ];

      await pg.query(q, v);

      const uwq = "insert into user_watch (user_id, watch_id) values ($1, $2)";
      const uwv = [userId, id];
      await pg.query(uwq, uwv);

      await pg.query("commit");
      const watch = await this.getWatch(id);

      return watch;
    } finally {
      await pg.query("rollback");
      pg.release();
    }
  }

  async removeUserFromWatch(watchId: string, userId: string): Promise<void> {
    const deleteUserFromWatchQuery = `delete from user_watch where user_id = $1 and watch_id = $2`;
    const v = [
      userId,
      watchId,
    ];

    await this.pool.query(deleteUserFromWatchQuery, v);

    const downstreamWatchesQuery = `select id from watch where parent_watch_id = $1`;
    const downstreamWatchesQueryValues = [watchId];

    const result = await this.pool.query(downstreamWatchesQuery, downstreamWatchesQueryValues);

    if (result.rows.length > 0) {
      for (const downstreamWatch of result.rows) {
        await this.pool.query(deleteUserFromWatchQuery, [userId, downstreamWatch.id]);
      }
    }
  }

  async addUserToWatch(watchId: string, userId: string): Promise<void> {
    const insertUserWatchQuery = `insert into user_watch (user_id, watch_id) values ($1, $2)`;
    const v = [
      userId,
      watchId,
    ];

    await this.pool.query(insertUserWatchQuery, v);

    const downstreamWatchesQuery = `select id from watch where parent_watch_id = $1`;
    const downstreamWatchesQueryValues = [watchId];

    const result = await this.pool.query(downstreamWatchesQuery, downstreamWatchesQueryValues);

    if (result.rows.length > 0) {
      for (const downstreamWatch of result.rows) {
        await this.pool.query(insertUserWatchQuery, [ userId, downstreamWatch.id]);
      }
    }
  }

  async listWatchContributors(id: string): Promise<Contributor[]> {
    const q = `
      SELECT ship_user.id as user_id, ship_user.created_at, github_user.github_id, github_user.username, github_user.avatar_url
      FROM user_watch
        JOIN ship_user ON ship_user.id = user_watch.user_id
        LEFT OUTER JOIN github_user ON github_user.user_id = ship_user.id
      WHERE watch_id = $1
    `;
    const v = [id];

    const result = await this.pool.query(q, v);
    const contributors: Contributor[] = [];
    for (const row of result.rows) {
      const result = this.mapContributor(row);
      contributors.push(result);
    }
    return contributors;
  }

  private mapGeneratedFile(row: any): GeneratedFile {
    return {
      watchId: row.watch_id,
      createdAt: row.created_at,
      sequence: row.sequence,
      filepath: row.filepath,
    };
  }

  private mapContributor(row: any): Contributor {
    return {
      id: row.user_id,
      createdAt: row.created_at,
      githubId: row.github_id,
      login: row.username,
      avatar_url: row.avatar_url,
    };
  }

  private mapWatchVersion(row: any): Version {
    return {
      title: row.version_label,
      status: row.status,
      createdOn: row.created_at,
      sequence: row.sequence,
      pullrequestNumber: row.pullrequest_number,
      deployedAt: row.deployed_at,
    };
  }
}
