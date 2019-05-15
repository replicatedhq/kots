export const Healthz = `
type Healthz {
  version: String
}
`;

export const Query = `
  type Query {
    healthz: Healthz!
    ping: String!

    installationOrganizations(page: Int): GetInstallationsResponse
    orgRepos(org: String!, page: Int): GetForOrgResponse
    repoBranches(owner: String! repo:String! page: Int): [GetBranchesResponse]
    userInfo: UserInfo
    userFeatures: [Feature]
    orgMembers(org: String!, page: Int): [GetMembersResponse]

    listClusters: [Cluster]

    listPendingInitSessions: [PendingInitSession]

    listWatches: [Watch]
    searchWatches(watchName: String!): [Watch]
    getWatch(slug: String, id: String): Watch
    watchContributors(id: String!): [Contributor]
    getWatchVersion(id: String!, sequence: Int): VersionDetail

    listPendingWatchVersions(watchId: String!): [Version]
    listPastWatchVersions(watchId: String!): [Version]
    getCurrentWatchVersion(watchId: String!): Version

    validateUpstreamURL(upstream: String!): Boolean!

    listNotifications(watchId: String!): [Notification]
    getNotification(notificationId: String!): Notification
    pullRequestHistory(notificationId: String!): [PullRequestHistory]

    imageWatches(batchId: String!): [ImageWatch]

    getGitHubInstallationId: String!
  }
`;
