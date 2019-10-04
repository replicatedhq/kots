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
    isConfigurable: Boolean
  }
`;

const KotsAppLink = `
  type KotsAppLink {
    title: String
    uri: String
  }
`;

const KotsDownstream = `
  type KotsDownstream {
    name: String
    currentVersion: KotsVersion
    pendingVersions: [KotsVersion]
    pastVersions: [KotsVersion]
    cluster: Cluster
    links: [KotsAppLink]
  }
`;

const KotsVersion = `
  type KotsVersion {
    title: String!
    status: String!
    createdOn: String!
    sequence: Int
    releaseNotes: String
    deployedAt: String
    preflightResult: String
    preflightResultCreatedAt: String
  }
`;

const KotsAppMetadata = `
  type KotsAppMetadata {
    name: String!
    iconUri: String!
    namespace: String!
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

const AirgapInstallStatus = `
  type AirgapInstallStatus {
    installStatus: String
    currentMessage: String
  }
`;

const KotsConfigChildItem = `
  type KotsConfigChildItem {
    name: String
    title: String
    recommended: Boolean
    default: String
    value: String
  }
`;

const KotsConfigChildItemInput = `
  input KotsConfigChildItemInput {
    name: String
    title: String
    recommended: Boolean
    default: String
    value: String
  }
`;

const KotsConfigItem = `
  type KotsConfigItem {
    name: String
    type: String
    title: String
    helpText: String
    recommended: Boolean
    default: String
    value: String
    multiValue: [String]
    readOnly: Boolean
    writeOnce: Boolean
    when: String
    multiple: Boolean
    hidden: Boolean
    position: Int
    affix: String
    required: Boolean
    items: [KotsConfigChildItem]
  }
`;

const KotsConfigItemInput = `
  input KotsConfigItemInput {
    name: String
    type: String
    title: String
    helpText: String
    recommended: Boolean
    default: String
    value: String
    multiValue: [String]
    readOnly: Boolean
    writeOnce: Boolean
    when: String
    multiple: Boolean
    hidden: Boolean
    position: Int
    affix: String
    required: Boolean
    items: [KotsConfigChildItemInput]
  }
`;

const KotsConfigGroup = `
  type KotsConfigGroup {
    name: String!
    title: String
    description: String
    items: [KotsConfigItem]
  }
`;

const KotsConfigGroupInput = `
  input KotsConfigGroupInput {
    name: String!
    title: String
    description: String
    items: [KotsConfigItemInput]
  }
`;

export default [
  KotsApp,
  KotsAppLink,
  KotsDownstream,
  KotsVersion,
  KotsAppMetadata,

  AppRegistryDetails,
  KotsAppRegistryDetails,
  AirgapInstallStatus,

  KotsConfigChildItem,
  KotsConfigChildItemInput,
  KotsConfigItem,
  KotsConfigItemInput,
  KotsConfigGroup,
  KotsConfigGroupInput,
];
