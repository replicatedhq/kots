const Cluster = `
type Cluster {
  id: ID
  title: String
  slug: String
  lastUpdated: String
  createdOn: String
  shipOpsRef: ShipOpsRef
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

export default [
  Cluster,
  ShipOpsRef,
];
