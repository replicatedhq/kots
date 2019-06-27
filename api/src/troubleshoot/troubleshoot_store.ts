import * as pg from "pg";
import * as randomstring from "randomstring";
import * as _ from "lodash";
import { Params } from "../server/params";
import { signPutRequest, signGetRequest } from "../util/persistence/s3";
import { ReplicatedError } from "../server/errors";
import { Collector, SupportBundle, SupportBundleInsight, SupportBundleStatus } from "./";
import { parseWatchName } from "../watch";

export class TroubleshootStore {
  constructor(
    private readonly pool: pg.Pool,
    private readonly params: Params,
  ) {
  }

  public async getPreferedWatchCollector(watchId: string): Promise<Collector> {
    const q = `select release_collector, updated_collector, release_collector_updated_at, updated_collector_updated_at, use_updated_collector
    from watch_troubleshoot_collector where watch_id = $1`;
    const v = [watchId];

    const result = await this.pool.query(q, v);
    if (result.rowCount === 0) {
      return this.getDefaultCollector();
    }

    const row = result.rows[0];

    let releaseIsNewer: boolean = false;
    if (row.updated_collector_updated_at) {
      releaseIsNewer = new Date(row.updated_collector_updated_at) < new Date(row.release_collector_updated_at);
    };

    let collector: Collector = new Collector();
    if (row.use_updated_collector || !releaseIsNewer) {
      collector.spec = row.updated_collector;
    } else {
      collector.spec = row.release_collector;
    }

    return collector;
  }

  public getDefaultCollector(): Collector {
    const collector: Collector = new Collector();
    collector.spec = `TODO`;

    return collector;
  }

  public async getSupportBundle(id: string): Promise<SupportBundle> {
    const q = `select supportbundle.id, supportbundle.slug,
        supportbundle.watch_id,
        supportbundle.name,
        supportbundle.size,
        supportbundle.status,
        supportbundle.tree_index,
        supportbundle.created_at,
        supportbundle.uploaded_at,
        supportbundle.is_archived,
        supportbundle.analysis_id,
        supportbundle_analysis.error AS analysis_error,
        supportbundle_analysis.max_severity AS analysis_max_severity,
        supportbundle_analysis.insights_marshalled AS analysis_insights_marshalled,
        supportbundle_analysis.created_at AS analysis_created_at,
        watch.slug as watch_slug,
        watch.title as watch_title
      from supportbundle
        inner join watch on supportbundle.watch_id = watch.id
        left join supportbundle_analysis on supportbundle.analysis_id = supportbundle_analysis.id
      where supportbundle.id = $1`;
    const v = [id];
    const result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`Support Bundle ${id} not found`);
    }

    const row = result.rows[0];
    const supportBundle = new SupportBundle();

    supportBundle.id = row.id;
    supportBundle.slug = row.slug;
    supportBundle.watchId = row.watch_id;
    supportBundle.name = row.name;
    supportBundle.size = row.size;
    supportBundle.status = row.status;
    supportBundle.treeIndex = row.tree_index;
    supportBundle.createdAt = row.created_at;
    supportBundle.uploadedAt = row.uploaded_at;
    supportBundle.isArchived = row.is_archived;
    supportBundle.analysis = {
      id: row.analysis_id,
      error: row.analysis_error,
      maxSeverity: row.analysis_max_severity,
      insights: this.mapSupportBundleInsights(row.analysis_insights_marshalled),
      createdAt: row.analysis_created_at,
    };
    supportBundle.watchSlug = row.watch_slug;
    supportBundle.watchName = parseWatchName(row.watch_title);

    return supportBundle;
  }

  public async listSupportBundles(watchId: string): Promise<SupportBundle[]> {
    const q = `select id from supportbundle where watch_id = $1 order by created_at`;
    const v = [watchId];
    const result = await this.pool.query(q, v);
    const supportBundles: SupportBundle[] = [];
    for (const row of result.rows) {
      supportBundles.push(await this.getSupportBundle(row.id));
    }
    return supportBundles;
  }

  public async createSupportBundle(watchId: string, size: number): Promise<SupportBundle> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    const createdAt = new Date();
    const status: SupportBundleStatus = "pending";

    const q = `insert into supportbundle (id, slug, watch_id, size, status, created_at) values ($1, $2, $3, $4, $5, $6)`;
    let v = [
      id,
      id, // TODO: slug
      watchId,
      size,
      status,
      createdAt,
    ]
    await this.pool.query(q, v);
    return await this.getSupportBundle(id);
  }

  public async markSupportBundleUploaded(id: string): Promise<SupportBundle> {
    const status: SupportBundleStatus = "uploaded";

    const q = `update supportbundle set status = $2, uploaded_at = $3 where id = $1`;
    const v = [id, status, new Date()];
    await this.pool.query(q, v);
    return await this.getSupportBundle(id);
  }

  private mapSupportBundleInsights(insightsMarshalled: string): SupportBundleInsight[] {
    const insights: SupportBundleInsight[] = [];
    if (!insightsMarshalled) {
      return insights;
    }
    return JSON.parse(insightsMarshalled) as SupportBundleInsight[]
  }

  public async signSupportBundlePutRequest(supportBundle: SupportBundle): Promise<string> {
    if (status !== "pending") {
      throw new ReplicatedError(`Unable to generate signed put request for a support bundle in status ${supportBundle.status}`);
    }

    return signPutRequest(this.params.shipOutputBucket, `supportbundles/${supportBundle.id}/supportbundle.tar.gz`, "application/tar+gzip");
  }

  public async signSupportBundleGetRequest(supportBundle: SupportBundle): Promise<string> {
    if (status === "pending") {
      throw new ReplicatedError(`Unable to generate signed get request for a support bundle in status ${supportBundle.status}`);
    }

    return await signGetRequest(this.params.shipOutputBucket, `supportbundles/${supportBundle.id}/supportbundle.tar.gz`);
  }
}
