const KotsAppUpload = `
  type KotsAppUpload {
    hasPreflight: Boolean
    isAirgap: Boolean
    needsRegistry: Boolean
    slug: String
    isConfigurable: Boolean
  }
`;

const KotsApp = `
  type KotsApp {
    id: String
    name: String
    iconUri: String
    createdAt: String
    updatedAt: String
    slug: String
    upstreamUri: String
    currentSequence: Int
    lastUpdateCheckAt: String
    bundleCommand: String
    isAirgap: Boolean
    downstreams: [KotsDownstream]
    currentVersion: KotsVersion
    hasPreflight: Boolean
    isConfigurable: Boolean
    allowRollback: Boolean
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

const KotsGitOpsInput = `
  input KotsGitOpsInput {
    provider: String
    uri: String
    owner: String
    branch: String
    path: String
    format: String
    action: String
    otherServiceName: String
  }
`;

const KotsVersion = `
  type KotsVersion {
    title: String!
    status: String!
    createdOn: String!
    sequence: Int
    parentSequence: Int
    releaseNotes: String
    deployedAt: String
    source: String
    diffSummary: String
    preflightResult: String
    preflightResultCreatedAt: String
    hasError: Boolean
  }
`;

const KotsAppMetadata = `
  type KotsAppMetadata {
    name: String!
    iconUri: String!
    namespace: String!
    isKurlEnabled: Boolean
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

const KotsDownstreamOutput = `
  type KotsDownstreamOutput {
    dryrunStdout: String
    dryrunStderr: String
    applyStdout: String
    applyStderr: String
  }
`;

const ResourceState = `
  type ResourceState {
    kind: String!
    name: String!
    namespace: String!
    state: String!
  }
`;

const KotsAppStatus = `
  type KotsAppStatus {
    appId: String!
    updatedAt: String!
    resourceStates: [ResourceState!]!
    state: String!
  }
`;

const KotsAppDashboard = `
type KotsAppDashboard {
  appStatus: KotsAppStatus
  metrics: [MetricChart]
  prometheusAddress: String
}
`

export default [
  KotsAppUpload,
  KotsApp,
  KotsAppLink,
  KotsDownstream,
  KotsVersion,
  KotsAppMetadata,
  KotsDownstreamOutput,
  ResourceState,

  KotsGitOpsInput,

  AppRegistryDetails,
  KotsAppRegistryDetails,
  AirgapInstallStatus,

  KotsConfigChildItem,
  KotsConfigChildItemInput,
  KotsConfigItem,
  KotsConfigItemInput,
  KotsConfigGroup,
  KotsConfigGroupInput,

  KotsAppStatus,
  KotsAppDashboard,
];
