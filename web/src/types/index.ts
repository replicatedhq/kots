export type App = {
  allowRollback: Object | undefined;
  allowSnapshots: boolean;
  autoDeploy: string;
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
  entitlements: Entitlement[];
  expiresAt: string;
  id: string;
  isAirgapSupported: boolean;
  isGeoaxisSupported: boolean;
  isGitOpsSupported: boolean;
  isIdentityServiceSupported: boolean;
  isSemverRequired: boolean;
  isSnapshotSupported: boolean;
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
};

export type Credentials = {
  username: string;
  password: string;
};

export type DashboardResponse = {
  appStatus: AppStatus | null;
  metrics: Chart[];
  prometheusAddress: string;
};

export type Downstream = {
  cluster: Cluster;
  pastVersions: Object;
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
  channelId: string;
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
  preflightStatus: string;
  preflightResultCreatedAt: string;
  preflightSkipped: boolean;
  releaseNotes: string;
  semver: string;
  sequence: number;
  source: string;
  status: VersionStatus;
  title: string;
  updateCursor: string;
  versionLabel?: string;
  yamlErrors: string[];
  isChecked: boolean;
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
