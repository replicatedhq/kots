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
    isAirgap: Boolean
    downstreams: [KotsDownstream]
    currentVersion: KotsVersion
    hasPreflight: Boolean
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
    preflightResult: String
    preflightResultUpdatedAt: String
  }
`;

const KotsAppMetadata = `
  type KotsAppMetadata {
    name: String!
    iconUri: String!
  }
`;

const AppRegistryDetails = `
  input AppRegistryDetails {
    appSlug: String!
    hostname: String!
    username: String!
    password: String!
    namespace: String!
  }
`;

const KotsAppRegistryDetails = `
  type KotsAppRegistryDetails {
    registryHostname: String
    registryUsername: String
    registryPassword: String
    namespace: String
    lastSyncedAt: String
  }
`;

const AirgapUploadProgress = `
  type AirgapUploadProgress {
    value: Int
  }
`;

export default [
  KotsApp,
  KotsDownstream,
  KotsVersion,
  KotsAppMetadata,
  AppRegistryDetails,
  KotsAppRegistryDetails,
  AirgapUploadProgress
];
