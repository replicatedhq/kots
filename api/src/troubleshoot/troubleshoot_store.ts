import pg from "pg";
import randomstring from "randomstring";
import _ from "lodash";
import { Params } from "../server/params";
import { signPutRequest, signGetRequest } from "../util/s3";
import { ReplicatedError } from "../server/errors";
import { Collector, SupportBundle, SupportBundleInsight, SupportBundleStatus } from "./";
import { parseWatchName } from "../watch";
import { SupportBundleAnalysis } from "./supportbundle";

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

    let useUpdatedCollector = false;
    if (row.use_updated_collector) {
      useUpdatedCollector = true;
    }

    if (row.updated_collector_updated_at) {
      useUpdatedCollector = new Date(row.updated_collector_updated_at) > new Date(row.release_collector_updated_at);
    };

    let collector: Collector = new Collector();
    if (useUpdatedCollector) {
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
        supportbundle_analysis.id as analysis_id,
        supportbundle_analysis.error AS analysis_error,
        supportbundle_analysis.max_severity AS analysis_max_severity,
        supportbundle_analysis.insights AS analysis_insights,
        supportbundle_analysis.created_at AS analysis_created_at,
        watch.slug as watch_slug,
        watch.title as watch_title
      from supportbundle
        inner join watch on supportbundle.watch_id = watch.id
        left join supportbundle_analysis on supportbundle.id = supportbundle_analysis.supportbundle_id
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

    if (row.analysis_insights) {
      const insights: SupportBundleInsight[] = [];

      var marsheledInsights;
      try {
        marsheledInsights = JSON.parse(row.analysis_insights);
      } catch(err) {
        marsheledInsights = [];
      }

      for (const marshaledInsight of marsheledInsights) {
        const insight = new SupportBundleInsight();
        insight.key = marshaledInsight.name;
        insight.severity = marshaledInsight.severity;
        insight.primary = marshaledInsight.insight.primary;
        insight.detail = marshaledInsight.insight.detail;
        insight.icon = marshaledInsight.labels.icon;
        insight.iconKey = marshaledInsight.labels.iconKey;
        insight.desiredPosition = marshaledInsight.labels.desiredPosition;

        insights.push(insight);
      }

      const analysis = new SupportBundleAnalysis();
      analysis.id = row.analysis_id;
      analysis.error = row.analysis_error,
      analysis.maxSeverity = row.analysis_max_severity,
      analysis.insights = insights;
      analysis.createdAt = row.analysis_created_at,
      supportBundle.analysis = analysis;
    }

    supportBundle.watchSlug = row.watch_slug;
    supportBundle.watchName = parseWatchName(row.watch_title);

    return supportBundle;
  }

  public async assignTreeIndex(id: string, index: string): Promise<boolean> {
    const q = `update supportbundle set tree_index = $1 where id = $2`;

    const v = [index, id];
    const result = await this.pool.query(q, v);

    return result.rows.length !== 0;
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
    const result = await this.pool.query(q, v);
    if (result.rowCount === 0) {
      // TODO
    }
    return await this.getSupportBundle(id);
  }

  public async signSupportBundlePutRequest(supportBundle: SupportBundle): Promise<string> {
    if (supportBundle.status !== "pending") {
      throw new ReplicatedError(`Unable to generate signed put request for a support bundle in status ${supportBundle.status}`);
    }

    return signPutRequest(this.params, this.params.shipOutputBucket, `supportbundles/${supportBundle.id}/supportbundle.tar.gz`, "application/tar+gzip");
  }

  public async signSupportBundleGetRequest(supportBundle: SupportBundle): Promise<string> {
    if (supportBundle.status === "pending") {
      throw new ReplicatedError(`Unable to generate signed get request for a support bundle in status ${supportBundle.status}`);
    }

    return await signGetRequest(this.params, this.params.shipOutputBucket, `supportbundles/${supportBundle.id}/supportbundle.tar.gz`);
  }
}
