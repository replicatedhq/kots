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
    searchPendingInitSessions(title: String!): [PendingInitSession]
    getPendingIniSession(id: String!): PendingInitSession

    listHelmCharts: [HelmChart]
    getHelmChart(id: String!): HelmChart

    listWatches: [Watch]
    searchWatches(watchName: String!): [Watch]
    getWatch(slug: String, id: String): Watch
    getParentWatch(id: String): Watch

    getWatchLicense(watchId: String!, entitlementSpec: String): License
    getLatestWatchLicense(licenseId: String!, entitlementSpec: String): License

    watchContributors(id: String!): [Contributor]
    getWatchVersion(id: String!, sequence: Int): VersionDetail
    getDownstreamHistory(slug: String!): [VersionDetail]

    listPendingWatchVersions(watchId: String!): [Version]
    listPastWatchVersions(watchId: String!): [Version]
    getCurrentWatchVersion(watchId: String!): Version

    validateUpstreamURL(upstream: String!): Boolean!

    listNotifications(watchId: String!): [Notification]
    getNotification(notificationId: String!): Notification
    pullRequestHistory(notificationId: String!): [PullRequestHistory]

    imageWatches(batchId: String!): [ImageWatch]

    getGitHubInstallationId: String!

    watchCollectors(watchId: String!): TroubleshootCollectSpec
    listSupportBundles(watchSlug: String!): [SupportBundle]
    getSupportBundle(watchSlug: String!): SupportBundle
    supportBundleFiles(bundleId: ID!, fileNames: [String!]): String
  }
`;
