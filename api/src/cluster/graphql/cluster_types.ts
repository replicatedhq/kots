const Cluster = `
type Cluster {
  id: ID
  title: String
  slug: String
  lastUpdated: String
  createdOn: String
  shipOpsRef: ShipOpsRef
  watchCounts: WatchCounts
  totalApplicationCount: Int
  enabled: Boolean
  currentVersion: KotsVersion
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
  ShipOpsRef,
  WatchCounts,
];
