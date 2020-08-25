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
    getKotsApp(slug: String!): KotsApp
    getKotsAppDashboard(slug: String!, clusterId: String): KotsAppDashboard
    getKotsDownstreamHistory(clusterSlug: String!, upstreamSlug: String!): [KotsVersion]
    listPendingKotsVersions(slug: String!): [KotsVersion]
    listPastKotsVersions(slug: String!): [KotsVersion]
    getCurrentKotsVersion(slug: String!): KotsVersion

    listHelmCharts: [HelmChart]
    getHelmChart(id: String!): HelmChart

    getAppLicense(appId: String!): KLicense

    getFiles(slug: String!, sequence: Int!, fileNames: [String!]): String

    getKotsDownstreamOutput(appSlug: String!, clusterSlug: String!, sequence: Int!): KotsDownstreamOutput

    listSupportBundles(watchSlug: String!): [SupportBundle]
    getSupportBundle(watchSlug: String!): SupportBundle

    getPreflightCommand(appSlug: String, clusterSlug: String, sequence: String): String

    getAirgapInstallStatus: InstallStatus
    getOnlineInstallStatus: InstallStatus
    getImageRewriteStatus: ImageRewriteStatus

    kurl: Kurl

    getPrometheusAddress: String

    snapshotConfig(slug: String!): SnapshotConfig
  }
`;
