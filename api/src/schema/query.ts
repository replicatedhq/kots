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

    getKotsMetadata: KotsAppMetadata
    listApps: Apps
    getKotsApp(slug: String!): KotsApp
    getKotsAppDashboard(slug: String!, clusterId: String): KotsAppDashboard
    listDownstreamsForApp(slug: String!): [Cluster]
    getKotsDownstreamHistory(clusterSlug: String!, upstreamSlug: String!): [KotsVersion]
    listPendingKotsVersions(slug: String!): [KotsVersion]
    listPastKotsVersions(slug: String!): [KotsVersion]
    getCurrentKotsVersion(slug: String!): KotsVersion
    getAppRegistryDetails(slug: String!): KotsAppRegistryDetails

    listHelmCharts: [HelmChart]
    getHelmChart(id: String!): HelmChart

    listWatches: [Watch]
    searchWatches(watchName: String!): [Watch]
    getWatch(slug: String, id: String): Watch
    getParentWatch(id: String, slug: String): Watch

    getWatchLicense(watchId: String!): License
    getLatestWatchLicense(licenseId: String!): License

    getAppLicense(appId: String!): KLicense
    hasLicenseUpdates(appSlug: String!): Boolean!

    watchContributors(id: String!): [Contributor]
    getWatchVersion(id: String!, sequence: Int): VersionDetail
    getDownstreamHistory(slug: String!): [VersionDetail]

    getApplicationTree(slug: String!, sequence: Int!): String
    getFiles(slug: String!, sequence: Int!, fileNames: [String!]): String

    getKotsApplicationTree(slug: String!, sequence: Int!): String
    getKotsFiles(slug: String!, sequence: Int!, fileNames: [String!]): String
    getKotsConfigGroups(slug: String!, sequence: Int!): [KotsConfigGroup]
    getKotsDownstreamOutput(appSlug: String!, clusterSlug: String!, sequence: Int!): KotsDownstreamOutput
    getConfigForGroups(slug: String!, sequence: Int!, configGroups: [KotsConfigGroupInput]!): [KotsConfigGroup]

    listPendingWatchVersions(watchId: String!): [Version]
    listPastWatchVersions(watchId: String!): [Version]
    getCurrentWatchVersion(watchId: String!): Version

    validateUpstreamURL(upstream: String!): Boolean!

    validateRegistryInfo(endpoint: String, username: String, password: String, org: String): String!

    listNotifications(watchId: String!): [Notification]
    getNotification(notificationId: String!): Notification
    pullRequestHistory(notificationId: String!): [PullRequestHistory]

    getGitHubInstallationId: String!

    watchCollectors(watchId: String!): TroubleshootCollectSpec
    listKotsSupportBundles(kotsSlug: String!): [SupportBundle]
    listSupportBundles(watchSlug: String!): [SupportBundle]
    getSupportBundle(watchSlug: String!): SupportBundle
    supportBundleFiles(bundleId: ID!, fileNames: [String!]): String
    getSupportBundleCommand(watchSlug: String): String

    listPreflightResults(watchId: String, slug: String): [PreflightResult]
    getKotsPreflightResult(appSlug: String!, clusterSlug: String!, sequence: Int!): PreflightResult
    getLatestKotsPreflightResult: PreflightResult

    getAirgapInstallStatus: AirgapInstallStatus

    kurl: Kurl

    getPrometheusAddress: String
  }
`;
