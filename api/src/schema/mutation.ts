const Mutation = `
type Mutation {
  ping: String
  createGithubNonce: String!
  createGithubAuthToken(state: String!, code: String!): AccessToken
  trackScmLead(deploymentPreference: String!, emailAddress: String!, scmProvider: String!): String
  loginToAdminConsole(password: String!): AdminSignupInfo
  logout: String

  createShipOpsCluster(title: String!): Cluster
  createGitOpsCluster(title: String!, installationId: Int, gitOpsRef: GitOpsRefInput): Cluster
  updateCluster(clusterId: String!, clusterName: String!, gitOpsRef: GitOpsRefInput): Cluster
  deleteCluster(clusterId: String!): Boolean

  collectSupportBundle(appId: String, clusterId: String): Boolean

  createKotsDownstream(appId: String!, clusterId: String!): Boolean
  deleteKotsDownstream(slug: String!, clusterId: String!): Boolean
  deleteKotsApp(slug: String!): Boolean
  checkForKotsUpdates(appId: ID!): Boolean
  uploadKotsLicense(value: String!): KotsAppUpload
  deployKotsVersion(upstreamSlug: String!, sequence: Int!, clusterSlug: String!): Boolean
  updateRegistryDetails(registryDetails: AppRegistryDetails!): Boolean
  resumeInstallOnline(slug: String!): KotsApp
  updateAppConfig(slug: String!, sequence: Int!, configGroups: [KotsConfigGroupInput]!, createNewVersion: Boolean): Boolean
  updateKotsApp(appId: String!, appName: String, iconUri: String): Boolean

  createWatch(stateJSON: String!): Watch
  updateWatch(watchId: String!, watchName: String, iconUri: String): Watch
  deleteWatch(watchId: String!, childWatchIds: [String]): Boolean
  updateStateJSON(slug: String!, stateJSON: String!): Watch
  deployWatchVersion(watchId: String!, sequence: Int): Boolean
  addWatchContributor(watchId: ID!, githubId: Int!, login: String!, avatarUrl: String): [Contributor]
  removeWatchContributor(watchId: ID!, contributorId: String!): [Contributor]
  checkForUpdates(watchId: ID!): Boolean
  syncWatchLicense(watchId: String!, licenseId: String!): License

  createNotification(watchId: String!, webhook: WebhookNotificationInput, email: EmailNotificationInput): Notification
  updateNotification(watchId: String!, notificationId: String!, webhook: WebhookNotificationInput, email: EmailNotificationInput): Notification
  enableNotification(watchId: String!, notificationId: String!, enabled: Int!): Notification
  deleteNotification(id: String!, isPending: Boolean): Boolean

  createInitSession(pendingInitId: String, upstreamUri: String, clusterID: String, githubPath: String): InitSession!
  createUnforkSession(upstreamUri: String!, forkUri: String!): UnforkSession!
  createUpdateSession(watchId: ID!): UpdateSession!
  createEditSession(watchId: ID!): EditSession!

  drainNode(name: String): Boolean
  deleteNode(name: String): Boolean
}
`;

export const all = [Mutation];
