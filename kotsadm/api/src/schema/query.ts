export const Healthz = `
type Healthz {
  version: String
}
`;

export const Query = `
  type Query {
    healthz: Healthz!
    ping: String!

    userFeatures: [Feature]

    listClusters: [Cluster]

    listApps: Apps
    getGitOpsRepo: KotsGitOps
    getKotsApp(slug: String!): KotsApp
    getKotsAppDashboard(slug: String!, clusterId: String): KotsAppDashboard
    listDownstreamsForApp(slug: String!): [Cluster]
    getKotsDownstreamHistory(clusterSlug: String!, upstreamSlug: String!): [KotsVersion]
    listPendingKotsVersions(slug: String!): [KotsVersion]
    listPastKotsVersions(slug: String!): [KotsVersion]
    getCurrentKotsVersion(slug: String!): KotsVersion

    listHelmCharts: [HelmChart]
    getHelmChart(id: String!): HelmChart

    getAppLicense(appId: String!): KLicense

    getFiles(slug: String!, sequence: Int!, fileNames: [String!]): String

    getAppConfigGroups(slug: String!, sequence: Int!): [KotsConfigGroup]
    getKotsDownstreamOutput(appSlug: String!, clusterSlug: String!, sequence: Int!): KotsDownstreamOutput
    templateConfigGroups(slug: String!, sequence: Int!, configGroups: [KotsConfigGroupInput]!): [KotsConfigGroup]

    listSupportBundles(watchSlug: String!): [SupportBundle]
    getSupportBundle(watchSlug: String!): SupportBundle

    getPreflightCommand(appSlug: String, clusterSlug: String, sequence: String): String

    getAirgapInstallStatus: InstallStatus
    getOnlineInstallStatus: InstallStatus
    getImageRewriteStatus: ImageRewriteStatus
    getUpdateDownloadStatus: UpdateDownloadStatus

    kurl: Kurl

    getPrometheusAddress: String

    snapshotConfig(slug: String!): SnapshotConfig
    snapshotDetail(slug: String!, id: String!): SnapshotDetail
    restoreDetail(appId: String!, restoreName: String!): RestoreDetail
  }
`;
