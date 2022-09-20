export type App = {
  allowSnapshots: boolean;
  chartPath: string;
  credentials: Credentials;
  currentSequence: number;
  downstream: DownStream;
  hasPreflight: boolean;
  isConfigurable: boolean;
  isGeoaxisSupported: boolean;
  isGitOpsSupported: boolean;
  isIdentityServiceSupported: boolean;
  name: string;
  namespace: string;
  needsRegistry?: boolean;
  slug: string;
};

export type Credentials = {
  username: string;
  password: string;
};

export type DownStream = {
  currentVersion: Version;
  gitops: GitOps;
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

export type Version = {
  parentSequence: number;
  semver: string;
  sequence: number;
  versionLabel?: string;
};
