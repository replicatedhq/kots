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
    isGitOpsSupported: Boolean
    allowRollback: Boolean
    kubectlVersion: String
    kustomizeVersion: String
    allowSnapshots: Boolean
    licenseType: String
    updateCheckerSpec: String
  }
`;

const KotsGitOps = `
  type KotsGitOps {
    enabled: Boolean
    provider: String
    uri: String
    path: String
    branch: String
    hostname: String
    format: String
    action: String
    deployKey: String
    isConnected: Boolean
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
    gitops: KotsGitOps
  }
`;

const KotsGitOpsInput = `
  input KotsGitOpsInput {
    provider: String
    uri: String
    owner: String
    branch: String
    hostname: String
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
    commitUrl: String
    gitDeployable: Boolean
    hasError: Boolean
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

const InstallStatus = `
  type InstallStatus {
    installStatus: String
    currentMessage: String
  }
`;

const ImageRewriteStatus = `
  type ImageRewriteStatus {
    status: String
    currentMessage: String
  }
`;

const UpdateDownloadStatus = `
  type UpdateDownloadStatus {
    status: String
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
    help_text: String
    recommended: Boolean
    default: String
    value: String
    error: String
    data: String
    multi_value: [String]
    readonly: Boolean
    write_once: Boolean
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
    help_text: String
    recommended: Boolean
    default: String
    value: String
    error: String
    data: String
    multi_value: [String]
    readonly: Boolean
    write_once: Boolean
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
    renderError: String
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
`;

export default [
  KotsApp,
  KotsAppLink,
  KotsDownstream,
  KotsVersion,
  KotsDownstreamOutput,
  ResourceState,

  KotsGitOps,
  KotsGitOpsInput,

  AppRegistryDetails,
  KotsAppRegistryDetails,
  InstallStatus,
  ImageRewriteStatus,
  UpdateDownloadStatus,

  KotsConfigChildItem,
  KotsConfigChildItemInput,
  KotsConfigItem,
  KotsConfigItemInput,
  KotsConfigGroup,
  KotsConfigGroupInput,

  KotsAppStatus,
  KotsAppDashboard,
];
