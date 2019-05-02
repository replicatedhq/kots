const ClusterItem = `
  type ClusterItem {
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

export const types = [
  ClusterItem,
  GitOpsRef,
  GitOpsRefInput,
  ShipOpsRef,
  WatchCounts,
];
