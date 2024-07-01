// This is ReponseApp in the go types
export type App = {
  allowRollback: Object | undefined;
  allowSnapshots: boolean;
  autoDeploy: string;
  chartPath: string;
  createdAt: string;
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
  isSupportBundleUploadSupported: boolean;
  licenseType: string;
  name: string;
  namespace: string;
  needsRegistry?: boolean;
  slug: string;
  updateCheckerSpec: string;
  appState: string;
};

export type AppLicense = {
  assignee: string;
  channelName: string;
  entitlements: Entitlement[];
  expiresAt: string;
  id: string;
  isAirgapSupported: boolean;
  isGeoaxisSupported: boolean;
  isGitOpsSupported: boolean;
  isIdentityServiceSupported: boolean;
  isSemverRequired: boolean;
  isSnapshotSupported: boolean;
  isDisasterRecoverySupported: boolean;
  isSupportBundleUploadSupported: boolean;
  lastSyncedAt: string;
  licenseSequence: number;
  licenseType: string;
  changingLicense: boolean;
  entitlementsToShow: string[];
  isViewingLicenseEntitlements: boolean;
  licenseChangeFile: LicenseFile | null;
  licenseChangeMessage: string;
  licenseChangeMessageType: string;
  loading: boolean;
  message: string;
  messageType: string;
  showLicenseChangeModal: boolean;
  showNextStepModal: boolean;
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
  state?: string;
  isUpgrading?: boolean;
};

export type Credentials = {
  username: string;
  password: string;
};

export type DashboardResponse = {
  appStatus: AppStatus | null;
  metrics: Chart[];
  prometheusAddress: string;
  embeddedClusterState: string;
};

export type Downstream = {
  cluster: Cluster;
  pastVersions: Version[];
  currentVersion: Version;
  gitops: GitOps;
  links: DashboardActionLink[];
  pendingVersions: Version[];
};

export type GitOps = {
  provider: string;
  isConnected: boolean;
  uri: string;
};

export type KotsParams = {
  downstreamSlug?: string;
  firstSequence: string | undefined;
  owner: string;
  redactorSlug: string;
  secondSequence: string | undefined;
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
  isEmbeddedCluster: boolean;
};

export type PreflightError = {
  error: string;
  isRbac: boolean;
};

export type PreflightResult = {
  appSlug: string;
  clusterSlug: string;
  createdAt: string;
  hasFailingStrictPreflights: boolean;
  result: string;
  sequence: number;
  skipped: boolean;
};

export type PreflightProgress = {
  completedCount: number;
  currentName: string;
  currentStatus: string;
  totalCount: number;
  updatedAt: string;
};

export type PreflightResultResponse = {
  errors?: PreflightError[];
  results?: PreflightResult[];
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

export type SupportBundle = {
  analysis: SupportBundleAnalysis;
  createdAt: string;
  id: string;
  isArchived: boolean;
  name: string;
  sharedAt: string;
  size: number;
  slug: string;
  status: string;
  uploadedAt: string;
};

type SupportBundleAnalysis = {
  createdAt: string;
  insights: SupportBundleInsight[];
};

export type SupportBundleInsight = {
  detail: string;
  icon: string;
  iconKey: string;
  key: string;
  primary: string;
  severity: string;
};

export type SupportBundleProgress = {
  collectorCount: number;
  collectorsCompleted: number;
  message: string;
};

export type AvailableUpdate = {
  versionLabel: string;
  updateCursor: string;
  channelId: string;
  isRequired: boolean;
  upstreamReleasedAt: string;
  releaseNotes: string;
  isDeployable: boolean;
  nonDeployableCause: string;
};

export type Version = {
  channelId: string;
  commitUrl: string;
  createdOn: string;
  deployedAt: string;
  diffSummary: string;
  diffSummaryError: string;
  downloadStatus: VersionDownloadStatus;
  gitDeployable: boolean;
  hasConfig: boolean;
  isChecked: boolean;
  isDeployable: boolean;
  isRequired: boolean;
  needsKotsUpgrade: boolean;
  nonDeployableCause: string;
  parentSequence: number;
  preflightResult: string;
  preflightResultCreatedAt: string;
  preflightSkipped: boolean;
  preflightStatus: string;
  releaseNotes: string;
  semver: string;
  sequence: number;
  source: string;
  status: VersionStatus;
  appTitle: string;
  appIconUri: string;
  updateCursor: string;
  upstreamReleasedAt: string;
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
  | "pending"
  | "pending_cluster_management"
  | "pending_config"
  | "pending_download"
  | "pending_preflight"
  | "waiting"
  | "unknown";

export type LicenseFile = {
  preview: string;
  lastModified: number;
  lastModifiedDate: Date;
  name: string;
  size: number;
  type: string;
  webkitRelativePath: string;
};

export type Series = {
  data: { timestamp: number; value: number }[];
  legendTemplate: string;
  metric: {
    name: string;
    value: string;
  }[];
};

export type Chart = {
  title: string | null | undefined;
  tickTemplate: string;
  tickFormat: string;
  series: Series[];
};

export type Snapshot = {
  name: string;
  status: string;
  trigger: string;
  sequence: number;
  startedAt: string;
  finishedAt: string;
  expiresAt: string;
  volumeCount: number;
  volumeSuccessCount: number;
  volumeBytes: number;
  volumeSizeHuman: string;
};

export type SnapshotSettings = {
  store: {
    aws: {
      region: string;
      bucket: string;
      accessKeyId: string;
      secretAccessKey: string;
    };
    azure: { accountName: string; accountKey: string; container: string };
    gcp: { bucket: string; projectId: string; serviceAccountKey: string };
    other: {
      endpoint: string;
      bucket: string;
      accessKeyId: string;
      secretAccessKey: string;
    };
    internal: { bucket: string };
    nfs: { server: string; path: string };
    hostpath: { path: string };
    bucket: string;
    path: string;
    fileSystem: string;
  };
  fileSystemConfig: {
    nfs: { server: string; path: string };
    hostPath: { path: string };
  };
};
