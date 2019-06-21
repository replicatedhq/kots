const TroubleshootCollectSpec = `
type TroubleshootCollectSpec {
  spec: String
}`;

const TroubleshootAnalyzeSpec = `
type TroubleshootAnalyzeSpec {
  spec: String
}
`;

const SupportBundle = `
type SupportBundle {
  id: ID!
  slug: String!
  watchId: String!
  name: String
  size: Int!
  notes: String
  status: String!
  resolution: String
  treeIndex: String
  viewed: Boolean!
  createdAt: String!
  uploadedAt: String
  isArchived: Boolean

  analysis: SupportBundleAnalysis

  watchSlug: String!
  watchName: String!
}`;

const SupportBundleAnalysis = `
type SupportBundleAnalysis {
  id: ID!
  error: String
  maxSeverity: String
  insights: [SupportBundleInsight]
  createdAt: String
}`;

const SupportBundleInsight = `
type SupportBundleInsight {
  key: String!
  severity: String!
  primary: String!
  detail: String
  icon: String
  icon_key: String
  desiredPosition: Int
  labels: [Label]
}`;

const Label = `
type Label {
  key: String!
  value: String
}`;

const SupportBundleUpload = `
type SupportBundleUpload {
  uploadUri: String!
  supportBundle: SupportBundle
}`;

export default [
  TroubleshootCollectSpec,
  TroubleshootAnalyzeSpec,
  SupportBundle,
  SupportBundleAnalysis,
  SupportBundleInsight,
  Label,
  SupportBundleUpload,
];
