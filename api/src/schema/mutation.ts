const Mutation = `
type Mutation {
  ping: String
  loginToAdminConsole(password: String!): AdminSignupInfo
  logout: String

  createShipOpsCluster(title: String!): Cluster
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
  updateAppConfig(slug: String!, sequence: Int!, configGroups: [KotsConfigGroupInput]!, createNewVersion: Boolean): Boolean
  updateKotsApp(appId: String!, appName: String, iconUri: String): Boolean

  setAppGitOps(appId: String!, clusterId: String!, gitOpsInput: KotsGitOpsInput!): String
  syncAppLicense(appSlug: String!, airgapLicense: String): KLicense

  drainNode(name: String): Boolean
  deleteNode(name: String): Boolean
  generateWorkerAddNodeCommand: Command!
  generateMasterAddNodeCommand: Command!

  setPrometheusAddress(value: String!): Boolean
  deletePrometheusAddress: Boolean
}
`;

export const all = [Mutation];
