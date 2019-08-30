const KotsApp = `
  type KotsApp {
    id: String
    name: String
    iconUri: String
    createdAt: String
    updatedAt: String
    slug: String
    currentSequence: Int
    lastUpdateCheckAt: String
    bundleCommand: String
    downstreams: [KotsDownstream]
    currentVersion: KotsVersion
  }
`;

const KotsDownstream = `
  type KotsDownstream {
    name: String
    currentVersion: KotsVersion
    pendingVersions: [KotsVersion]
    pastVersions: [KotsVersion]
    cluster: Cluster
  }
`;

const KotsVersion = `
  type KotsVersion {
    title: String!
    status: String!
    createdOn: String!
    sequence: Int
    deployedAt: String
  }
`;

const KotsAppMetadata = `
  type KotsAppMetadata {
    name: String!
    iconUri: String!
  }
`;

export default [
  KotsApp,
  KotsDownstream,
  KotsVersion,
  KotsAppMetadata,
];
