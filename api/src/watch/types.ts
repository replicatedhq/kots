const WatchItem = `
  type WatchItem {
    id: ID
    stateJSON: String
    watchName: String
    slug: String
    watchIcon: String
    lastUpdated: String
    createdOn: String
    contributors: [ContributorItem]
    notifications: [Notification]
    features: [Feature]
    cluster: Cluster
    watches: [WatchItem]
    currentVersion: VersionItem
    pendingVersions: [VersionItem]
    pastVersions: [VersionItem]
    currentVersion: VersionItem
    parentWatch: WatchItem
  }
`;

const VersionItem = `
  type VersionItem {
    title: String!
    status: String!
    createdOn: String!
    sequence: Int
    pullrequestNumber: Int
  }
`

const VersionItemDetail = `
  type VersionItemDetail {
    title: String
    status: String
    createdOn: String
    sequence: Int
    pullrequestNumber: Int
    rendered: String
  }
`
const StateMetadata = `
  type StateMetadata {
    name: String
    icon: String
    version: String
  }
`;

const ContributorItem = `
  type ContributorItem {
    id: ID
    createdAt: String
    githubId: Int
    login: String
    avatar_url: String
  }
`;

const ContributorItemInput = `
  input ContributorItemInput {
    githubId: Int
    login: String
    avatar_url: String
  }
`;

export const types = [
  WatchItem,
  StateMetadata,
  ContributorItem,
  ContributorItemInput,
  VersionItem,
  VersionItemDetail,
];
