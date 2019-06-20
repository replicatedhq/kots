import * as pg from "pg";
import { ReplicatedError } from "../server/errors";
import { SupportBundle, SupportBundleAnalysis, SupportBundleInsight } from "./types";
import { parseWatchName } from "../watch";

const supportBundleFields = [
  "supportbundle.id",
  "supportbundle.slug",
  "supportbundle.watch_id",
  "supportbundle.name",
  "supportbundle.size",
  "supportbundle.notes",
  "supportbundle.status",
  "supportbundle.uri",
  "supportbundle.resolution",
  "supportbundle.tree_index",
  "supportbundle.viewed",
  "supportbundle.created_at",
  "supportbundle.uploaded_at",
  "supportbundle.is_archived",
];

const supportBundleAnalysisFields = [
  "supportbundle_analysis.id AS analysis_id",
  "supportbundle_analysis.error AS analysis_error",
  "supportbundle_analysis.max_severity AS analysis_max_severity",
  "supportbundle_analysis.insights_marshalled AS analysis_insights_marshalled",
  "supportbundle_analysis.created_at AS analysis_created_at",
];

export class TroubleshootStore {
  constructor(
    private readonly pool: pg.Pool,
  ) {
  }

  async getSupportBundle(id: string): Promise<SupportBundle> {
    const q = `SELECT
        ${supportBundleFields}, ${supportBundleAnalysisFields},
        watch.slug as watch_slug, watch.title as watch_title
      FROM supportbundle
        INNER JOIN watch on supportbundle.watch_id = watch.id
        LEFT JOIN supportbundle_analysis on supportbundle.id = supportbundle_analysis.supportbundle_id
      WHERE supportbundle.id = ?
        AND supportbundle_analysis.id = (
          SELECT sa2.id
          FROM supportbundle_analysis AS sa2
          WHERE sa2.supportbundle_id = ?
          ORDER BY sa2.created_at DESC
          LIMIT 1
        )`;
    const v = [id, id];
    const result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`Support Bundle ${id} was not found`);
    }
    return this.mapSupportBundle(result.rows[0]);
  }

  private mapSupportBundle(row: any): SupportBundle {
    const supportBundle: SupportBundle = {
      id: row.id,
      slug: row.slug,
      watchId: row.watch_id,
      name: row.name,
      size: row.size,
      notes: row.notes,
      status: row.status,
      uri: row.uri,
      resolution: row.resolution,
      treeIndex: row.tree_index,
      viewed: row.viewed,
      createdAt: row.created_at,
      uploadedAt: row.uploaded_at,
      isArchived: row.is_archived,
      signedUri: "TODO",
      analysis: {
        id: row.analysis_id,
        error: row.analysis_error,
        maxSeverity: row.analysis_max_severity,
        insights: this.mapSupportBundleInsights(row.analysis_insights_marshalled),
        createdAt: row.analysis_created_at,
      },
      watchSlug: row.watch_slug,
      watchName: parseWatchName(row.watch_title),
    };
    return supportBundle;
  }

  private mapSupportBundleInsights(insightsMarshalled: string): [SupportBundleInsight] {

  }
}
