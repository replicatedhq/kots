export const Healthz = `
type Healthz {
  version: String
}
`;

export const Query = `
  type Query {
    healthz: Healthz!
    ping: String!

    userInfo: UserInfo
    userFeatures: [Feature]

    listClusters: [Cluster]

    getKotsMetadata: KotsAppMetadata
    listApps: Apps
    getGitOpsRepo: KotsGitOps
    getKotsApp(slug: String!): KotsApp
    getKotsAppDashboard(slug: String!, clusterId: String): KotsAppDashboard
    getKotsLicenseType(slug: String!): String
    listDownstreamsForApp(slug: String!): [Cluster]
    getKotsDownstreamHistory(clusterSlug: String!, upstreamSlug: String!): [KotsVersion]
    listPendingKotsVersions(slug: String!): [KotsVersion]
    listPastKotsVersions(slug: String!): [KotsVersion]
    getCurrentKotsVersion(slug: String!): KotsVersion
    getAppRegistryDetails(slug: String!): KotsAppRegistryDetails

    listHelmCharts: [HelmChart]
    getHelmChart(id: String!): HelmChart

    getAppLicense(appId: String!): KLicense

    getFiles(slug: String!, sequence: Int!, fileNames: [String!]): String

    getKotsApplicationTree(slug: String!, sequence: Int!): String
    getKotsFiles(slug: String!, sequence: Int!, fileNames: [String!]): String
    getAppConfigGroups(slug: String!, sequence: Int!): [KotsConfigGroup]
    getKotsDownstreamOutput(appSlug: String!, clusterSlug: String!, sequence: Int!): KotsDownstreamOutput
    templateConfigGroups(slug: String!, sequence: Int!, configGroups: [KotsConfigGroupInput]!): [KotsConfigGroup]

    validateRegistryInfo(slug: String, endpoint: String, username: String, password: String, org: String): String!

    listKotsSupportBundles(kotsSlug: String!): [SupportBundle]
    listSupportBundles(watchSlug: String!): [SupportBundle]
    getSupportBundle(watchSlug: String!): SupportBundle
    supportBundleFiles(bundleId: ID!, fileNames: [String!]): String
    getSupportBundleCommand(watchSlug: String): String

    getKotsPreflightResult(appSlug: String!, clusterSlug: String!, sequence: Int!): PreflightResult
    getLatestKotsPreflightResult: PreflightResult
    getPreflightCommand(appSlug: String, clusterSlug: String, sequence: String): String

    getAirgapInstallStatus: AirgapInstallStatus
    getImageRewriteStatus: ImageRewriteStatus
    getUpdateDownloadStatus: UpdateDownloadStatus

    kurl: Kurl

    getPrometheusAddress: String
  }
`;
