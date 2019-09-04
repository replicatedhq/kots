import pg from "pg";
import { Params } from "../server/params";
import { KotsApp, KotsVersion, KotsAppRegistryDetails } from "./";
import { ReplicatedError } from "../server/errors";
import { signPutRequest, signGetRequest } from "../util/s3";
import randomstring from "randomstring";
import slugify from "slugify";
import _ from "lodash";

export class KotsAppStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) {}

  async listClusterIDsForApp(id: string): Promise<string[]> {
    const q = `select cluster_id from app_downstream where app_id = $1`;
    const v = [
      id,
    ];

    const result = await this.pool.query(q, v);
    const clusterIds: string[] = [];
    for (const row of result.rows) {
      clusterIds.push(row.cluster_id);
    }

    return clusterIds;
  }

  async listAppsForCluster(clusterId: string): Promise<KotsApp[]> {
    const q = `select app_id from app_downstream where cluster_id = $1`;
    const v = [
      clusterId,
    ];

    const result = await this.pool.query(q, v);
    const apps: KotsApp[] = [];
    for (const row of result.rows) {
      apps.push(await this.getApp(row.app_id));
    }

    return apps;
  }

  async createDownstream(appId: string, downstreamName: string, clusterId: string): Promise<void> {
    const q = `insert into app_downstream (app_id, downstream_name, cluster_id) values ($1, $2, $3)`;
    const v = [
      appId,
      downstreamName,
      clusterId,
    ];

    await this.pool.query(q, v);
  }

  async createMidstreamVersion(id: string, sequence: number, versionLabel: string, updateCursor: string, supportBundleSpec: string | undefined, preflightSpec: string | undefined): Promise<void> {
    const q = `insert into app_version (app_id, sequence, created_at, version_label, update_cursor, supportbundle_spec, preflight_spec)
      values ($1, $2, $3, $4, $5, $6, $7)`;
    const v = [
      id,
      sequence,
      new Date(),
      versionLabel,
      updateCursor,
      supportBundleSpec,
      preflightSpec,
    ];

    await this.pool.query(q, v);

    const qq = `update app set current_sequence = $1 where id = $2`;
    const vv = [
      sequence,
      id,
    ];

    await this.pool.query(qq, vv);
  }

  async createDownstreamVersion(id: string, parentSequence: number, clusterId: string, versionLabel: string): Promise<void> {
    let q = `select max(sequence) as last_sequence from app_downstream_version where app_id = $1 and cluster_id = $2`;
    let v: any[] = [
      id,
      clusterId,
    ];
    const result = await this.pool.query(q, v);

    const newSequence = result.rows[0].last_sequence !== null ? parseInt(result.rows[0].last_sequence) + 1 : 0;

    q = `insert into app_downstream_version (app_id, cluster_id, sequence, parent_sequence, created_at, version_label, status)
      values ($1, $2, $3, $4, $5, $6, $7)`;
    v = [
      id,
      clusterId,
      newSequence,
      parentSequence,
      new Date(),
      versionLabel,
      "pending"
    ];

    await this.pool.query(q, v);
  }

  async listPastVersions(appId: string, clusterId: string): Promise<KotsVersion[]> {
    let q = `select current_sequence from app where id = $1`;
    let v = [
      appId,
    ];

    let result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`No past versions found`);
    }
    const sequence = result.rows[0].current_sequence;

    // If there is not a current_sequence, then there can't be past versions
    if (sequence === null) {
      return [];
    }

    q = `select created_at, version_label, status, sequence, applied_at from app_downstream_version where app_id = $1 and cluster_id = $3 and sequence < $2 order by sequence desc`;
    v = [
      appId,
      sequence,
      clusterId
    ];

    result = await this.pool.query(q, v);
    const versionItems: KotsVersion[] = [];

    for (const row of result.rows) {
      versionItems.push(this.mapKotsAppVersion(row));
    }

    return versionItems;
  }

  async listPendingVersions(appId: string, clusterId: string): Promise<KotsVersion[]> {
    let q = `select current_sequence from app where id = $1`;
    let v = [
      appId,
    ];

    let result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`No pending versions found`);
    }
    let sequence = result.rows[0].current_sequence;

    // If there is not a current_sequence, then all versions are future versions
    if (sequence === null) {
      sequence = -1;
    }

    q = `
select created_at, version_label, status, sequence, applied_at
from app_downstream_version
where app_id = $1 and cluster_id = $3 and sequence > $2
order by sequence desc`;
    v = [
      appId,
      sequence,
      clusterId,
    ];

    result = await this.pool.query(q, v);
    const versionItems: KotsVersion[] = [];

    for (const row of result.rows) {
      versionItems.push(this.mapKotsAppVersion(row));
    }

    return versionItems;
  }

  async getCurrentDownstreamVersion(appId: string, clusterId: string): Promise<KotsVersion | undefined> {
    let q = `select current_sequence from app_downstream where app_id = $1 and cluster_id = $2`;
    let v = [
      appId,
      clusterId
    ];
    let result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`No current version found`);
    }
    const sequence = result.rows[0].current_sequence;

    if (sequence === null) {
      return;
    }

    q = `select created_at, version_label, status, sequence, applied_at from app_downstream_version where app_id = $1 and cluster_id = $3 and sequence = $2`;
    v = [
      appId,
      sequence,
      clusterId,
    ];

    result = await this.pool.query(q, v);

    const versionItem = this.mapKotsAppVersion(result.rows[0]);

    return versionItem;
  }

  async getCurrentAppVersion(appId: string): Promise<KotsVersion | undefined> {
    let q = `select current_sequence from app where id = $1`;
    let v = [
      appId,
    ];
    let result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`No current version found`);
    }
    const sequence = result.rows[0].current_sequence;

    if (sequence === null) {
      return;
    }

    q = `select created_at, version_label, status, sequence, applied_at from app_version where app_id = $1 and sequence = $2`;
    v = [
      appId,
      sequence,
    ];

    result = await this.pool.query(q, v);
    // TODO: check for row length here?
    const versionItem = this.mapKotsAppVersion(result.rows[0]);

    return versionItem;
  }

  async getMidstreamUpdateCursor(appId: string): Promise<string> {
    const q = `select update_cursor from app_version where app_id = $1 and sequence = (select current_sequence from app where id = $1)`;
    const v = [
      appId,
    ];

    const result = await this.pool.query(q, v);

    return result.rows[0].update_cursor;
  }

  async getCurrentVersion(appId: string, clusterId: string): Promise<KotsVersion | undefined> {
    let q = `select current_sequence from app where id = $1`;
    let v = [
      appId,
    ];
    let result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`No current version found`);
    }
    const sequence = result.rows[0].current_sequence;

    if (sequence === null) {
      return;
    }

    q = `select created_at, version_label, status, sequence, applied_at from app_downstream_version where app_id = $1 and cluster_id = $3 and sequence = $2`;
    v = [
      appId,
      sequence,
      clusterId,
    ];

    result = await this.pool.query(q, v);
    const versionItem = this.mapKotsAppVersion(result.rows[0]);

    return versionItem;
  }

  async deployVersion(appId: string, sequence: number, clusterId: string): Promise<void> {
    const q = `update app_downstream set current_sequence = $1 where app_id = $2 and cluster_id = $3`;
      const v = [
        sequence,
        appId,
        clusterId,
      ];
      await this.pool.query(q, v);
  }

  async getAppRegistryDetails(appId: string): Promise<KotsAppRegistryDetails> {
    const q = `select registry_hostname, registry_username, registry_password, namespace, last_registry_sync from app where id = $1`;
    const v = [
      appId
    ];
    const result = await this.pool.query(q, v);
    if (result.rowCount === 0) {
      throw new ReplicatedError(`Unable to get registry details for app with the ID of ${appId}`);
    }
    return this.mapAppRegistryDetails(result.rows[0]);
  }

  async updateRegistryDetails(appId: string, hostname: string, username: string, password: string, namespace: string): Promise<void> {
    const q = `update app set registry_hostname = $1, registry_username = $2, registry_password = $3, namespace = $4, last_registry_sync = $5 where id = $6`;
    const v = [
      hostname,
      username,
      password,
      namespace,
      new Date(),
      appId,
    ];
    await this.pool.query(q, v);
  }

  async listKotsApps(userId?: string): Promise<KotsApp[]> {
    const q = `select id from app inner join user_app on app_id = id where user_app.user_id = $1`;
    const v = [userId];

    const result = await this.pool.query(q, v);
    const apps: KotsApp[] = [];
    for (const row of result.rows) {
      apps.push(await this.getApp(row.id));
    }

    const qq = `select id from app where is_all_users = true`;
    const resultTwo = await this.pool.query(qq);
    for (const row of resultTwo.rows) {
      apps.push(await this.getApp(row.id));
    }

    return apps;
  }

  async deleteDownstream(appId: string, clusterId: string): Promise<Boolean> {
    const q = `delete from app_downstream where app_id = $1 and cluster_id = $2`;
    const v = [appId, clusterId];

    const result = await this.pool.query(q, v);
    if (result.rowCount === 0) {
      throw new ReplicatedError(`No downstreams with the id of ${clusterId} were found`);
    }

    return true;
  }

  async deleteApp(appId: string): Promise<Boolean> {
    const pg = await this.pool.connect();
    try {
      let q: string;
      const v = [appId];

      await pg.query("begin");
      q = `delete from user_app where app_id = $1`;
      await pg.query(q, v);

      q = `delete from app_version where app_id = $1`;
      await pg.query(q, v);

      q = `delete from app_downstream where app_id = $1`;
      await pg.query(q, v);

      q = `delete from app_downstream_version where app_id = $1`;
      await pg.query(q, v);

      q = `delete from app where id = $1`;
      await pg.query(q, v);

      await pg.query("commit");
    } finally {
      await pg.query("rollback");
      pg.release();
    }
    return true;
  }

  async getApp(id: string): Promise<KotsApp> {
    const q = `select id, name, license, upstream_uri, icon_uri, created_at, updated_at, slug, current_sequence, last_update_check_at from app where id = $1`;
    const v = [id];

    const result = await this.pool.query(q, v);

    if (result.rowCount == 0) {
      throw new ReplicatedError("not found");
    }

    const row = result.rows[0];

    const kotsApp = new KotsApp();
    kotsApp.id = row.id;
    kotsApp.name = row.name;
    kotsApp.license = row.license;
    kotsApp.upstreamUri = row.upstream_uri;
    kotsApp.iconUri = row.icon_uri;
    kotsApp.createdAt = new Date(row.created_at);
    kotsApp.updatedAt = row.updated_at ? new Date(row.updated_at) : undefined;
    kotsApp.slug = row.slug;
    kotsApp.currentSequence = row.current_sequence;
    kotsApp.lastUpdateCheckAt = row.last_update_check_at ? new Date(row.last_update_check_at) : undefined;
    kotsApp.bundleCommand = await kotsApp.getSupportBundleCommand(row.slug);
    return kotsApp;
  }

  async getIdFromSlug(slug: string): Promise<string> {
    const q = "select id from app where slug = $1";
    const v = [slug];

    const result = await this.pool.query(q, v);
    if (result.rowCount === 0) {
      throw new ReplicatedError(`Unable to find appId for slug ${slug}`);
    }
    return result.rows[0].id;
  }

  async createKotsApp(name: string, upstreamURI: string, license: string, userId?: string): Promise<KotsApp> {
    if (!name) {
      throw new Error("missing name");
    }

    const id = randomstring.generate({ capitalization: "lowercase" });
    const titleForSlug = name.replace(/\./g, "-");

    let slugProposal = slugify(titleForSlug, { lower: true });

    let i = 0;
    let foundUniqueSlug = false;
    while (!foundUniqueSlug) {
      if (i > 0) {
        slugProposal = `${slugify(titleForSlug, { lower: true })}-${i}`;
      }
      const qq = `select count(1) as count from app where slug = $1`;
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
      const q = `insert into app (id, name, icon_uri, created_at, slug, upstream_uri, license, is_all_users)
      values ($1, $2, $3, $4, $5, $6, $7, $8)`;
      const v = [
        id,
        name,
        "",
        new Date(),
        slugProposal,
        upstreamURI,
        license,
        !userId
      ];

      await pg.query(q, v);

      if (userId) { // unset user id means all users
        const uwq = "insert into user_app (user_id, app_id) values ($1, $2)";
        const uwv = [userId, id];
        await pg.query(uwq, uwv);
      }

      await pg.query("commit");
      const watch = await this.getApp(id);

      return watch;
    } finally {
      await pg.query("rollback");
      pg.release();
    }
  }

  private mapKotsAppVersion(row: any): KotsVersion {
    if (!row) {
      throw new ReplicatedError("No app provided to map function");
    }
    return {
      title: row.version_label,
      status: row.status || "",
      createdOn: row.created_at,
      sequence: row.sequence,
      deployedAt: row.applied_at,
    };
  }

  async getAirgapBundlePutUrl(filename: string): Promise<string> {
    const signed = await signPutRequest(this.params, this.params.airgapBucket, filename, "application/tar+gzip", 60);
    return signed
  }

  async getAirgapBundleGetUrl(filename: string): Promise<string> {
    const signed = await signGetRequest(this.params, this.params.airgapBucket, filename, 60);
    return signed
  }
  private mapAppRegistryDetails(row: any): KotsAppRegistryDetails {
    if (!row) {
      throw new ReplicatedError("No app provided to map function");
    }
    return {
      registryHostname: row.registry_hostname,
      registryUsername: row.registry_username,
      registryPassword: row.registry_password,
      namespace: row.namespace,
      lastSyncedAt: row.last_registry_sync,
    };
  }

}
