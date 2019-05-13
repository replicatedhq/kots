export interface Watch {
  id: string;
  stateJSON: string;
  watchName: string;
  slug: string;
  watchIcon: string;
  lastUpdated: string;
  createdOn: string;
  contributors: [Contributor]
  notifications: [Notification]
  // features: [Feature]
  // cluster: Cluster
  watches: [Watch]
  currentVersion: Version
  pendingVersions: [Version]
  pastVersions: [Version]
  parentWatch: Watch
}

export interface Version {
  title: string;
  status: string;
  createdOn: string;
  sequence: number;
  pullrequestNumber: number;
}

export interface VersionDetail {
  title: string;
  status: string;
  createdOn: string;
  sequence: number;
  pullrequestNumber: number;
  rendered: string;
}

export interface StateMetadata {
  name: string;
  icon: string;
  version: string;
}

export interface Contributor {
  id: string;
  createdAt: string;
  githubId: number;
  login: string;
  avatar_url: string;
}

export interface ContributorInput {
  githubId: number;
  login: string;
  avatar_url: string;
}
