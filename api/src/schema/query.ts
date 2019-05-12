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
    repoBranches(owner: String! repo:String! page: Int): [GetBranchesResponseItem]
    userInfo: UserInfo
    userFeatures: [Feature]
    orgMembers(org: String!, page: Int): [GetMembersResponseItem]

    listClusters: [Cluster]

    listWatches: [WatchItem]
    searchWatches(watchName: String!): [WatchItem]
    getWatch(slug: String, id: String): WatchItem
    watchContributors(id: String!): [ContributorItem]
    getWatchVersion(id: String!, sequence: Int): VersionItemDetail

    listPendingWatchVersions(watchId: String!): [VersionItem]
    listPastWatchVersions(watchId: String!): [VersionItem]
    getCurrentWatchVersion(watchId: String!): VersionItem

    validateUpstreamURL(upstream: String!): Boolean!

    listNotifications(watchId: String!): [Notification]
    getNotification(notificationId: String!): Notification
    pullRequestHistory(notificationId: String!): [PullRequestHistoryItem]

    imageWatchItems(batchId: String!): [ImageWatchItem]

    getGitHubInstallationId: String!
  }
`;
