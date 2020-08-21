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
  deployKotsVersion(upstreamSlug: String!, sequence: Int!, clusterSlug: String!): Boolean
  updateRegistryDetails(registryDetails: AppRegistryDetails!): Boolean
  updateDownstreamsStatus(slug: String!, sequence: Int!, status: String!): Boolean
  updateKotsApp(appId: String!, appName: String, iconUri: String): Boolean

  createGitOpsRepo(gitOpsInput: KotsGitOpsInput!): Boolean
  updateGitOpsRepo(gitOpsInput: KotsGitOpsInput!, uriToUpdate: String): Boolean
  updateAppGitOps(appId: String!, clusterId: String!, gitOpsInput: KotsGitOpsInput!): Boolean
  testGitOpsConnection(appId: String!, clusterId: String!): Boolean
  disableAppGitops(appId: String!, clusterId: String!): Boolean
  ignorePreflightPermissionErrors(appSlug: String, clusterSlug: String, sequence: Int): Boolean
  retryPreflights(appSlug: String, clusterSlug: String, sequence: Int): Boolean
  resetGitOpsData: Boolean

  setPrometheusAddress(value: String!): Boolean
  deletePrometheusAddress: Boolean

  saveSnapshotConfig(appId: String!, inputValue: Int!, inputTimeUnit: String!, schedule: String!, autoEnabled: Boolean!): Boolean
  deleteSnapshot(snapshotName: String!): Boolean
  cancelRestore(appId: String!): Boolean
}
`;

export const all = [Mutation];
