import _ from "lodash";

type SupportBundleStatus = "pending" | "uploaded" | "analyzing" | "analyzed" | "analysis_error";

export interface SupportBundleUpload {
  uploadUri: string;
  supportBundle: SupportBundle;
}

export class SupportBundle {
  id: string;
  slug: string;
  watchId: string;
  name: string;
  size: number;
  status: SupportBundleStatus;
  treeIndex: string;
  createdAt: Date;
  uploadedAt: Date;
  isArchived: boolean;
  analysis: SupportBundleAnalysis;
  watchSlug: string;
  watchName: string;
  kotsLicenseType?: string;

  public toSchema() {
    return {
      id: this.id,
      slug: this.slug,
      name: this.name,
      size: this.size,
      status: this.status,
      treeIndex: this.treeIndex,
      createdAt: this.createdAt ? this.createdAt.toISOString() : undefined,
      uploadedAt: this.uploadedAt ? this.uploadedAt.toISOString() : undefined,
      isArchived: this.isArchived,
      kotsLicenseType: this.kotsLicenseType,
      analysis: this.analysis ? this.analysis.toSchema() : undefined
    };
  }
};

export class SupportBundleAnalysis {
  id: string;
  error: string;
  maxSeverity: string;
  insights: SupportBundleInsight[];
  createdAt: Date;

  public toSchema() {
    return {
      id: this.id,
      error: this.error,
      maxSeverity: this.maxSeverity,
      insights: _.map(this.insights, (insight) => {
        return insight.toSchema();
      }),
      createdAt: this.createdAt ? this.createdAt.toISOString() : undefined,
    };
  }
};

export class SupportBundleInsight {
  key: string;
  severity: string;
  primary: string;
  detail: string;
  icon: string;
  iconKey: string;
  desiredPosition: number;

  public toSchema() {
    return {
      key: this.key,
      severity: this.severity,
      primary: this.primary,
      detail: this.detail,
      icon: this.icon,
      icon_key: this.iconKey,
      desiredPosition: this.desiredPosition,
    };
  }
}

