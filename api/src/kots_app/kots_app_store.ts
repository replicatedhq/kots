import pg from "pg";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import { KotsApp } from "./";
import { ReplicatedError } from "../server/errors";
import randomstring from "randomstring";
import slugify from "slugify";
import _ from "lodash";

export class KotsAppStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) {}

  async createDownstream(appId: string, downstreamName: string, clusterId: string): Promise<void> {
    const q = `insert into app_downstream (app_id, downstream_name, cluster_id) values ($1, $2, $3)`;
    const v = [
      appId,
      downstreamName,
      clusterId,
    ];

    await this.pool.query(q, v);
  }

  async createKotsAppVersion(id: string, sequence: number, versionLabel: string): Promise<void> {
    const q = `insert into app_version (app_id, sequence, created_at, version_label) values ($1, $2, $3, $4)`;
    const v = [
      id,
      sequence,
      new Date(),
      versionLabel,
    ];

    await this.pool.query(q, v);

    const qq = `update app set current_sequence = $1 where id = $2`;
    const vv = [
      sequence,
      id,
    ];

    await this.pool.query(qq, vv);
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
      await pg.query("begin");
      const q = `delete from user_app where app_id = $1`;
      const v = [appId];
      await pg.query(q, v);

      const qq = `delete from app_version where app_id = $1`;
      await pg.query(qq, v);

      const qqq = `delete from app_downstream where app_id = $1`;
      await pg.query(qqq, v);

      const qqqq = `delete from app where id = $1`;
      await pg.query(qqqq, v);

      await pg.query("commit");
    } finally {
      await pg.query("rollback");
      pg.release(); 
    }
    return true;
  }

  async getApp(id: string): Promise<KotsApp> {
    const q = `select id, name, icon_uri, created_at, updated_at, slug, current_sequence, last_update_check_at from app where id = $1`;
    const v = [id];

    const result = await this.pool.query(q, v);

    if (result.rowCount == 0) {
      throw new ReplicatedError("not found");
    }

    const row = result.rows[0];

    const kotsApp = new KotsApp();
    kotsApp.id = row.id;
    kotsApp.name = row.name;
    kotsApp.iconUri = row.icon_uri;
    kotsApp.createdAt = new Date(row.created_at);
    kotsApp.updatedAt = row.updated_at ? new Date(row.updated_at) : undefined;
    kotsApp.slug = row.slug;
    kotsApp.currentSequence = row.current_sequence;
    kotsApp.lastUpdateCheckAt = row.last_update_check_at ? new Date(row.last_update_check_at) : undefined;

    return kotsApp;
  }

  async getIdFromSlug(slug: string): Promise<string> {
    const q = "select id from app where slug = $1";
    const v = [slug];

    const result = await this.pool.query(q, v);
    return result.rows[0].id;
  }

  async createKotsApp(name: string, userId?: string): Promise<KotsApp> {
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
      const q = `insert into app (id, name, icon_uri, created_at, slug, is_all_users)
      values ($1, $2, $3, $4, $5, $6)`;
      const v = [
        id,
        name,
        "",
        new Date(),
        slugProposal,
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
}
