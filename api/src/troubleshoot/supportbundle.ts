
export type SupportBundleStatus = "pending" | "uploaded" | "analyzing" | "analyzed" | "analysis_error";

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

  public toSchema() {
    return {
      id: this.id,
      slug: this.slug,
      name: this.name,
      size: this.size,
      status: this.status,
      treeIndex: this.treeIndex,
      createdAt: this.createdAt.toISOString(),
      uploadedAt: this.uploadedAt.toISOString(),
      isArchived: this.isArchived,
    };
  }
};

export interface SupportBundleAnalysis {
  id: string;
  error: string;
  maxSeverity: string;
  insights: SupportBundleInsight[];
  createdAt: string;
};

export interface SupportBundleInsight {
  key: string;
  severity: string;
  primary: string;
  detail: string;
  icon: string;
  icon_key: string;
  desiredPosition: number;
  labels: Label[];
}

export interface Label {
  key: string;
  value: string;
}
