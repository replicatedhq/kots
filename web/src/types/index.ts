export type App = {
  allowSnapshots: boolean;
  autoDeploy: boolean;
  chartPath: string;
  credentials: Credentials;
  currentSequence: number;
  downstream: Downstream;
  hasPreflight: boolean;
  helmName: string;
  id: string;
  iconUri: string;
  isAirgap: boolean;
  isAppIdentityServiceSupported: boolean;
  isConfigurable: boolean;
  isGeoaxisSupported: boolean;
  isGitOpsSupported: boolean;
  isIdentityServiceSupported: boolean;
  isSemverRequired: boolean;
  name: string;
  namespace: string;
  needsRegistry?: boolean;
  slug: string;
  updateCheckerSpec: string;
};

export type AppLicense = {
  assignee: string;
  channelName: string;
  entitlements: string[];
  expiresAt: string;
  id: string;
  isAirgapSupported: boolean;
  isGeoaxisSupported: boolean;
  isGitOpsSupported: boolean;
  isIdentityServiceSupported: boolean;
  isSemverRequired: boolean;
  isSnapshotsSupported: boolean;
  isSupportBundleUploadSupported: boolean;
  lastSyncedAt: string;
  licenseSequence: number;
  licenseType: string;
};

type AppStatus = {
  appId: string;
  resourceStates: ResourceStates[];
  sequence: number;
  state: string;
  updatedAt: string;
};

export type AppStatusState =
  | "degraded"
  | "degrading"
  | "missing"
  | "ready"
  | "unavailable"
  | "updating";

type Cluster = {
  id: number;
  slug: string;
};

export type Credentials = {
  username: string;
  password: string;
};

export type DashboardResponse = {
  appStatus: AppStatus | null;
  metrics: string[];
  prometheusAddress: string;
};

export type Downstream = {
  cluster: Cluster;
  currentVersion: Version;
  gitops: GitOps;
  links: DashboardActionLink[];
  pendingVersions: Version[];
};

export type GitOps = {
  isConnected: true;
  provider: string;
  uri: string;
};

export type KotsParams = {
  owner: string;
  sequence: string;
  slug: string;
  tab: string;
};

export type DashboardActionLink = {
  title: string;
  uri: string;
};

export type Entitlement = {
  title: string;
  value: string;
  label: string;
  valueType: "Text" | "Boolean" | "Integer" | "String";
};

export type Metadata = {
  isAirgap: boolean;
  isKurl: boolean;
};

export type ThemeState = {
  navbarLogo: string | null;
};

export type ResourceStates = {
  kind: string;
  name: string;
  namespace: string;
  // from https://github.com/replicatedhq/kots/blob/84b7e4e0e9275bb200a36be69691c4944eb8cf8f/pkg/appstate/types/types.go#L10-L14
  state: AppStatusState;
};

export type Version = {
  commitUrl: string;
  createdOn: string;
  deployedAt: string;
  diffSummary: string;
  diffSummaryError: string;
  downloadStatus: VersionDownloadStatus;
  gitDeployable: boolean;
  isDeployable: boolean;
  isRequired: boolean;
  needsKotsUpgrade: boolean;
  nonDeployableCause: string;
  parentSequence: number;
  preflightResult: string;
  preflightSkipped: boolean;
  releaseNotes: string;
  semver: string;
  sequence: number;
  source: string;
  status: VersionStatus;
  title: string;
  versionLabel?: string;
  yamlErrors: string[];
};

export type VersionDownloadStatus = {
  downloadingVersion: boolean;
  downloadingVersionMessage: string;
  downloadingVersionError?: boolean;
  message?: string;
  status?: VersionStatus;
};

export type VersionStatus =
  | "deployed"
  | "deploying"
  | "failed"
  | "merged"
  | "pending"
  | "pending_config"
  | "pending_download"
  | "pending_preflight"
  | "waiting";
