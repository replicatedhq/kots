export type App = {
  allowSnapshots: boolean;
  chartPath: string;
  credentials: Credentials;
  currentSequence: number;
  downstream: Downstream;
  hasPreflight: boolean;
  iconUri: string;
  isAirgap: boolean;
  isConfigurable: boolean;
  isGeoaxisSupported: boolean;
  isGitOpsSupported: boolean;
  isIdentityServiceSupported: boolean;
  name: string;
  namespace: string;
  needsRegistry?: boolean;
  slug: string;
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
  isIdentityServiceSupported: boolean;j
  isSemverRequired: boolean;
  isSnapshotsSupported: boolean;
  isSupportBundleUploadSupported: boolean;
  lastSyncedAt: string;
  licenseSequence: number;
  licenseType: string;
}

export type Credentials = {
  username: string;
  password: string;
};

export type Dashboard = {
  appStatus: {
    appId: string;
    resourceStates: {
      kind: string;
      name: string;
      namespace: string;
      // from https://github.com/replicatedhq/kots/blob/84b7e4e0e9275bb200a36be69691c4944eb8cf8f/pkg/appstate/types/types.go#L10-L14
      state: "ready" | "updating" | "degrading" | "unavailable" | "missing";
    }[];
    sequence: number;
    state: string;
    updatedAt: string;
  } | null;
  metrics: string[];
  prometheusAddress: string;
}

export type Downstream = {
  currentVersion: Version;
  gitops: GitOps;
  links: DashboardActionLink[];
  pendingVersions: Version[];
};

export type GitOps = {
  isConnected: true;
  uri: string;
};

export type KotsParams = {
  sequence: string;
  slug: string;
};

export type DashboardActionLink = {
  title: string;
  uri: string;
}

export type Version = {
  parentSequence: number;
  semver: string;
  sequence: number;
  versionLabel?: string;
};
