const Watch = `
  type Watch {
    id: ID
    stateJSON: String
    watchName: String
    slug: String
    watchIcon: String
    lastUpdated: String
    createdOn: String
    contributors: [Contributor]
    notifications: [Notification]
    features: [Feature]
    cluster: Cluster
    watches: [Watch]
    currentVersion: Version
    pendingVersions: [Version]
    pastVersions: [Version]
    currentVersion: Version
    parentWatch: Watch
    metadata: String
    config: [ConfigGroup]
    entitlements: [Entitlement]
    lastUpdateCheck: String
  }
`;

const Version = `
  type Version {
    title: String!
    status: String!
    createdOn: String!
    sequence: Int
    pullrequestNumber: Int
    deployedAt: String
  }
`

const VersionDetail = `
  type VersionDetail {
    title: String
    status: String
    createdOn: String
    sequence: Int
    pullrequestNumber: Int
    rendered: String
    deployedAt: String
  }
`
const StateMetadata = `
  type StateMetadata {
    name: String
    icon: String
    version: String
  }
`;

const Contributor = `
  type Contributor {
    id: ID
    createdAt: String
    githubId: Int
    login: String
    avatar_url: String
  }
`;

const ConfigItem = `
  type ConfigItem {
    name: String
    title: String
    default: String
    value: String
    type: String
  }
`;

const ConfigGroup = `
  type ConfigGroup {
    name: String!
    title: String
    description: String
    items: [ConfigItem]
  }
`;

export default [
  Watch,
  StateMetadata,
  Contributor,
  Version,
  VersionDetail,
  ConfigItem,
  ConfigGroup,
];
