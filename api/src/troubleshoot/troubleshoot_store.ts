import pg from "pg";
import randomstring from "randomstring";
import _ from "lodash";
import { Params } from "../server/params";
import { getFileInfo } from "../util/s3";
import { ReplicatedError } from "../server/errors";
import { Collector, SupportBundle, SupportBundleInsight, SupportBundleStatus } from "./";
import { SupportBundleAnalysis } from "./supportbundle";
import { Analyzer } from "./analyzer";
import { getLicenseType } from "../util/utilities";

export class TroubleshootStore {
  constructor(
    private readonly pool: pg.Pool,
    private readonly params: Params,
  ) {
  }

  static defaultSpec = `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Collector
metadata:
  name: defalt-collector
spec:
  collectors: []`

  public async tryGetAnalyzersForKotsApp(id: string): Promise<Analyzer | void> {
    const q = `select analyzer_spec from app_version
      inner join app on app_version.app_id = app.id and app_version.sequence = app.current_sequence
      where app.id = $1`;
    const v = [id];

    const result = await this.pool.query(q, v);

    let analyzer: Analyzer = new Analyzer();
    if (result.rowCount === 0) {
      return;
    }

    return result.rows[0].analyzer_spec;
  }

  public async tryGetCollectorForKotsSlug(slug: string): Promise<Collector | void> {
    const q = `select supportbundle_spec from app_version
      inner join app on app_version.app_id = app.id and app_version.sequence = app.current_sequence
      where app.slug = $1`;
    const v = [slug];

    const result = await this.pool.query(q, v);

    let collector: Collector = new Collector();
    if (result.rowCount === 0) {
      return;
    }

    return result.rows[0].supportbundle_spec;
  }

  public async setAnalysisResult(supportBundleId: string, insights: string): Promise<void> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    const q = `insert into supportbundle_analysis (id, supportbundle_id, error, max_severity, insights, created_at) values ($1, $2, null, null, $3, $4)`;
    const v = [
      id,
      supportBundleId,
      insights,
      new Date(),
    ];

    await this.pool.query(q, v);
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
        watch.title as watch_title,
        app.name as kots_app_title,
        app_version.kots_license as kots_license
      from supportbundle
        left join watch on supportbundle.watch_id = watch.id
        left join app_downstream on supportbundle.watch_id = app_downstream.app_id
        left join app on supportbundle.watch_id = app.id
        left join app_version on supportbundle.watch_id = app_version.app_id
        left join supportbundle_analysis on supportbundle.id = supportbundle_analysis.supportbundle_id
      where supportbundle.id = $1`;
    const v = [id];
    const result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`Support Bundle ${id} not found`);
    }

    const row = result.rows[0];
    const supportBundle = new SupportBundle();
    const kotsLicenseType = await getLicenseType(row.kots_license);

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
    supportBundle.kotsLicenseType = kotsLicenseType;

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

        // key is a not nullable field in gql, but because of the loose coupling of the release
        // process between vendor analyze specs and kots (we can't control), we need to ensure
        // that we backfill keys that the analyzer doesn't put in...

        // this should be fixed in future supportbundle ffi functions

        insight.key = marshaledInsight.name || randomstring.generate({ capitalization: "lowercase" });
        insight.severity = marshaledInsight.severity;
        insight.primary = marshaledInsight.insight.primary;
        insight.detail = marshaledInsight.insight.detail;
        insight.icon = marshaledInsight.labels.iconUri;
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
    supportBundle.watchName = row.kots_app_title;

    return supportBundle;
  }

  public async clearPendingSupportBundle(id: string): Promise<void> {
    const q = `delete from pending_supportbundle where id = $1`;
    const v = [id];

    await this.pool.query(q, v);
  }

  public async listPendingSupportBundlesForCluster(clusterId: string): Promise<any[]> {
    const q = `select id, app_id, cluster_id from pending_supportbundle where cluster_id = $1`;
    const v = [clusterId];

    const result = await this.pool.query(q, v);

    const pendingSupportBundles: any[] = [];
    for (const row of result.rows) {
      const pendingSupportBundle = {
        id: row.id,
        appId: row.app_id,
        clusterId: row.cluster_id,
      };

      pendingSupportBundles.push(pendingSupportBundle);
    }

    return pendingSupportBundles;
  }

  public async queueSupportBundleCollection(appId: string, clusterId: string): Promise<string> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    const q = `insert into pending_supportbundle (id, app_id, cluster_id, created_at) values ($1, $2, $3, $4)`;
    const v = [
      id,
      appId,
      clusterId,
      new Date(),
    ];

    await this.pool.query(q, v);

    return id;
  }

  public async supportBundleExists(id: string): Promise<boolean> {
    const q = `SELECT count(1) AS count FROM supportbundle WHERE id = $1`;
    const v = [id];
    const result = await this.pool.query(q, v);

    return result.rows[0].count === "1";
  }

  public async assignTreeIndex(id: string, index: string): Promise<boolean> {
    const q = `update supportbundle set tree_index = $1 where id = $2`;

    const v = [index, id];
    const result = await this.pool.query(q, v);

    return result.rows.length !== 0;
  }

  public async listSupportBundles(appOrWatchId: string): Promise<SupportBundle[]> {
    const q = `select id from supportbundle where watch_id = $1 order by created_at`;
    const v = [appOrWatchId];
    const result = await this.pool.query(q, v);
    const supportBundles: SupportBundle[] = [];
    for (const row of result.rows) {
      supportBundles.push(await this.getSupportBundle(row.id));
    }
    return supportBundles;
  }

  // creates a SupportBundle object which is not stored in DB, but can be used to
  // create signed upload URL
  public async getBlankSupportBundle(appOrWatchId: string): Promise<SupportBundle> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    const status: SupportBundleStatus = "pending";

    const supportBundle = new SupportBundle();
    supportBundle.id = id;
    supportBundle.watchId = appOrWatchId;
    supportBundle.status = status;

    return supportBundle;
  }

  public async createSupportBundle(appOrWatchId: string, size: number, id?: string): Promise<SupportBundle> {
    if (!id) {
      id = randomstring.generate({ capitalization: "lowercase" });
    }
    const createdAt = new Date();
    const status: SupportBundleStatus = "pending";

    const q = `insert into supportbundle (id, slug, watch_id, size, status, created_at) values ($1, $2, $3, $4, $5, $6)`;
    let v = [
      id,
      id, // TODO: slug
      appOrWatchId,
      size,
      status,
      createdAt,
    ]
    await this.pool.query(q, v);
    return await this.getSupportBundle(id);
  }

  public async markSupportBundleUploaded(id: string): Promise<void> {
    const status: SupportBundleStatus = "uploaded";

    const q = `update supportbundle set status = $2, uploaded_at = $3 where id = $1`;
    const v = [id, status, new Date()];
    await this.pool.query(q, v);
  }

  public async updateSupportBundleStatus(id: string, status: SupportBundleStatus): Promise<void> {
    const q = `update supportbundle set status = $2 where id = $1`;
    const v = [id, status];
    await this.pool.query(q, v);
  }

  async getSupportBundleCommand(watchSlug?: string): Promise<string> {
    let url = `API_ADDRESS/api/v1/troubleshoot`;
    if (watchSlug) {
      url = `${url}/${watchSlug}`;
    }
    const bundleCommand = `
    curl https://krew.sh/support-bundle | bash
    kubectl support-bundle ${url}
    `;
    return bundleCommand;
  }

  public async getSupportBundleFileInfo(supportBundleId: string): Promise<any> {
    const params = {
      Bucket: this.params.shipOutputBucket,
      Key: `supportbundles/${supportBundleId}/supportbundle.tar.gz`,
    };

    const info = await getFileInfo(this.params, params);
    return info
  }
}
