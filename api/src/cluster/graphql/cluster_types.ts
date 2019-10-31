const Cluster = `
type Cluster {
  id: ID
  title: String
  slug: String
  lastUpdated: String
  createdOn: String
  gitOpsRef: GitOpsRef
  shipOpsRef: ShipOpsRef
  watchCounts: WatchCounts
  totalApplicationCount: Int
  enabled: Boolean
  currentVersion: KotsVersion
}
`;

const GitOpsRef = `
  type GitOpsRef {
    owner: String!
    repo: String!
    branch: String
    path: String
  }
`;

const GitOpsRefInput = `
  input GitOpsRefInput {
    owner: String!
    repo: String!
    branch: String
  }
`;

const ShipOpsRef = `
  type ShipOpsRef {
    token: String!
  }
`;

const WatchCounts = `
  type WatchCounts {
    pending: Int
    past: Int
  }
`

export default [
  Cluster,
  GitOpsRef,
  GitOpsRefInput,
  ShipOpsRef,
  WatchCounts,
];
