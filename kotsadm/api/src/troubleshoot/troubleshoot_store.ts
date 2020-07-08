import pg from "pg";
import randomstring from "randomstring";
import _ from "lodash";
import { Params } from "../server/params";
import { ReplicatedError } from "../server/errors";
import { SupportBundle, SupportBundleInsight } from "./";
import { SupportBundleAnalysis } from "./supportbundle";
import { getLicenseType } from "../util/utilities";

export class TroubleshootStore {
  constructor(
    private readonly pool: pg.Pool,
    private readonly params: Params,
  ) {
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
        app.name as kots_app_title,
        app_version.kots_license as kots_license
      from supportbundle
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

  public async listSupportBundles(appOrWatchId: string): Promise<SupportBundle[]> {
    const q = `select id from supportbundle where watch_id = $1 order by created_at desc`;
    const v = [appOrWatchId];
    const result = await this.pool.query(q, v);
    const supportBundles: SupportBundle[] = [];
    for (const row of result.rows) {
      supportBundles.push(await this.getSupportBundle(row.id));
    }
    return supportBundles;
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
}
