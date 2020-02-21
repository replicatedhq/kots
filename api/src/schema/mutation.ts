const Mutation = `
type Mutation {
  ping: String
  logout: String

  updateCluster(clusterId: String!, clusterName: String!): Cluster
  deleteCluster(clusterId: String!): Boolean

  collectSupportBundle(appId: String, clusterId: String): Boolean

  createKotsDownstream(appId: String!, clusterId: String!): Boolean
  deleteKotsDownstream(slug: String!, clusterId: String!): Boolean
  deleteKotsApp(slug: String!): Boolean
  checkForKotsUpdates(appId: ID!): Int
  uploadKotsLicense(value: String!): KotsAppUpload
  deployKotsVersion(upstreamSlug: String!, sequence: Int!, clusterSlug: String!): Boolean
  updateRegistryDetails(registryDetails: AppRegistryDetails!): Boolean
  resumeInstallOnline(slug: String!): KotsApp
  updateDownstreamsStatus(slug: String!, sequence: Int!, status: String!): Boolean
  updateKotsApp(appId: String!, appName: String, iconUri: String): Boolean

  createGitOpsRepo(gitOpsInput: KotsGitOpsInput!): Boolean
  updateGitOpsRepo(gitOpsInput: KotsGitOpsInput!, uriToUpdate: String): Boolean
  updateAppGitOps(appId: String!, clusterId: String!, gitOpsInput: KotsGitOpsInput!): Boolean
  syncAppLicense(appSlug: String!, airgapLicense: String): KLicense
  testGitOpsConnection(appId: String!, clusterId: String!): Boolean
  disableAppGitops(appId: String!, clusterId: String!): Boolean
  ignorePreflightPermissionErrors(appSlug: String, clusterSlug: String, sequence: Int): Boolean
  retryPreflights(appSlug: String, clusterSlug: String, sequence: Int): Boolean
  resetGitOpsData: Boolean

  drainNode(name: String): Boolean
  deleteNode(name: String): Boolean

  setPrometheusAddress(value: String!): Boolean
  deletePrometheusAddress: Boolean

  saveSnapshotConfig(appId: String!, inputValue: Int!, inputTimeUnit: String!, schedule: String!, autoEnabled: Boolean!): Boolean
  snapshotProviderAWS(bucket: String!, prefix: String, region: String!, accessKeyID: String, accessKeySecret: String): Boolean
  snapshotProviderS3Compatible(bucket: String!, prefix: String, region: String!, endpoint: String!, accessKeyID: String, accessKeySecret: String): Boolean
  snapshotProviderAzure(bucket: String!, prefix: String, tenantID: String!, resourceGroup: String!, storageAccount: String!, subscriptionID: String!, clientID: String!, clientSecret: String!, cloudName: String!): Boolean
  snapshotProviderGoogle(bucket: String!, prefix: String, serviceAccount: String!): Boolean
  manualSnapshot(appId: String!): Boolean
  restoreSnapshot(snapshotName: String!): RestoreDetail
  deleteSnapshot(snapshotName: String!): Boolean
  cancelRestore(appId: String!): Boolean
}
`;

export const all = [Mutation];
