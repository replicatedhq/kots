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
  }
`;

const Version = `
  type Version {
    title: String!
    status: String!
    createdOn: String!
    sequence: Int
    pullrequestNumber: Int
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

const ContributorInput = `
  input ContributorInput {
    githubId: Int
    login: String
    avatar_url: String
  }
`;

export default [
  Watch,
  StateMetadata,
  Contributor,
  ContributorInput,
  Version,
  VersionDetail,
];
