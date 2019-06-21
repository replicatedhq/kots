import * as pg from "pg";
import * as randomstring from "randomstring";
import * as _ from "lodash";
import { Params } from "../server/params";
import { S3Signer } from "../util/persistence/s3";
import { ReplicatedError } from "../server/errors";
import { SupportBundle, SupportBundleInsight, SupportBundleStatus } from "./types";
import { parseWatchName } from "../watch";

export class TroubleshootStore {
  constructor(
    private readonly pool: pg.Pool,
    private readonly params: Params,
    private readonly s3Signer: S3Signer,
  ) {
  }

  public async getSupportBundle(id: string): Promise<SupportBundle> {
    const q = `SELECT
        supportbundle.id,
        supportbundle.slug,
        supportbundle.watch_id,
        supportbundle.name,
        supportbundle.size,
        supportbundle.notes,
        supportbundle.status,
        supportbundle.resolution,
        supportbundle.tree_index,
        supportbundle.viewed,
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
      FROM supportbundle
        INNER JOIN watch on supportbundle.watch_id = watch.id
        LEFT JOIN supportbundle_analysis on supportbundle.analysis_id = supportbundle_analysis.id
      WHERE supportbundle.id = $1`;
    const v = [id];
    const result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`Support Bundle ${id} not found`);
    }
    return await this.mapSupportBundle(result.rows[0]);
  }

  public async listSupportBundles(watchId: string): Promise<SupportBundle[]> {
    const q = `SELECT id
      FROM supportbundle
      WHERE watch_id = $1`;
    const v = [watchId];
    const result = await this.pool.query(q, v);
    const supportBundles: SupportBundle[] = [];
    for (const row of result.rows) {
      supportBundles.push(await this.getSupportBundle(row.id));
    }
    return _.sortBy(supportBundles, ["createdAt"]);
  }

  public async createSupportBundle(watchId: string, size: number, notes?: string): Promise<SupportBundle> {
    const id = randomstring.generate({ capitalization: "lowercase" });
    const createdAt = new Date();
    const status: SupportBundleStatus = "pending";

    const q = `INSERT INTO supportbundle
        (id, slug, watch_id, size, notes, status, created_at)
      VALUES ($1, $2, $3, $4, $5, $6, $7)`;
    let v = [
      id,
      id, // TODO: slug
      watchId,
      size,
      notes,
      status,
      createdAt,
    ]
    const result = await this.pool.query(q, v);
    return await this.getSupportBundle(id);
  }

  public async markSupportBundleUploaded(id: string): Promise<SupportBundle> {
    const status: SupportBundleStatus = "uploaded";

    const q = `UPDATE supportbundle
      SET status = $2, uploaded_at = $3
      WHERE id = $1`;
    const v = [id, status, new Date()];
    await this.pool.query(q, v);
    return await this.getSupportBundle(id);
  }

  public async getWatchIdFromToken(token: string): Promise<string> {
    const q = `SELECT watch_id
      FROM supportbundle_upload_token
      WHERE token = $1`;
    const v = [token];
    const result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`Watch id for token ${token} not found`);
    }
    return result.rows[0].watch_id;
  }

  private async mapSupportBundle(row: any): Promise<SupportBundle> {
    const supportBundle: SupportBundle = {
      id: row.id,
      slug: row.slug,
      watchId: row.watch_id,
      name: row.name,
      size: row.size,
      notes: row.notes,
      status: row.status,
      resolution: row.resolution,
      treeIndex: row.tree_index,
      viewed: row.viewed,
      createdAt: row.created_at,
      uploadedAt: row.uploaded_at,
      isArchived: row.is_archived,
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

  private mapSupportBundleInsights(insightsMarshalled: string): SupportBundleInsight[] {
    const insights: SupportBundleInsight[] = [];
    if (!insightsMarshalled) {
      return insights;
    }
    return JSON.parse(insightsMarshalled) as SupportBundleInsight[]
  }

  public async signSupportBundlePutRequest(supportBundle: SupportBundle): Promise<any> {
    if (status !== "pending") {
      throw new ReplicatedError(`Unable to generate signed put request for a support bundle in status ${supportBundle.status}`);
    }

    // NOTE: shipOutputBucket is awkward
    const signed = await this.s3Signer.signPutRequest({
      Bucket: this.params.shipOutputBucket,
      Key: `supportbundles/${supportBundle.id}/supportbundle.tar.gz`,
      ContentType: "application/tar+gzip",
    });
    return signed.signedUrl;
  }

  public async signSupportBundleGetRequest(supportBundle: SupportBundle): Promise<string | undefined> {
    if (status === "pending") {
      throw new ReplicatedError(`Unable to generate signed get request for a support bundle in status ${supportBundle.status}`);
    }

    // NOTE: shipOutputBucket is awkward
    const signed = await this.s3Signer.signGetRequest({
      Bucket: this.params.shipOutputBucket,
      Key: `supportbundles/${supportBundle.id}/supportbundle.tar.gz`,
    });
    return signed.signedUrl;
  }
}
