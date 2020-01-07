import gql from "graphql-tag";

export const createKotsDownstreamRaw = `
  mutation createKotsDownstream($appId: String!, $clusterId: String!) {
    createKotsDownstream(appId: $appId, clusterId: $clusterId)
  }
`;

export const createKotsDownstream = gql(createKotsDownstreamRaw);

export const deleteKotsDownstreamRaw = `
  mutation deleteKotsDownstream($slug: String!, $clusterId: String!) {
    deleteKotsDownstream(slug: $slug, clusterId: $clusterId)
  }
`;

export const deleteKotsDownstream = gql(deleteKotsDownstreamRaw);

export const deleteKotsAppRaw = `
  mutation deleteKotsApp($slug: String!) {
    deleteKotsApp(slug: $slug)
  }
`;

export const deleteKotsApp = gql(deleteKotsAppRaw);

export const checkForKotsUpdatesRaw = `
  mutation checkForKotsUpdates($appId: ID!) {
    checkForKotsUpdates(appId: $appId)
  }
`
export const checkForKotsUpdates = gql(checkForKotsUpdatesRaw);

export const uploadKotsLicenseRaw = `
  mutation uploadKotsLicense($value: String!) {
    uploadKotsLicense(value: $value) {
      hasPreflight
      slug
      isAirgap
      needsRegistry
      isConfigurable
    }
  }
`
export const uploadKotsLicense = gql(uploadKotsLicenseRaw);

export const deployKotsVersionRaw = `
  mutation deployKotsVersion($upstreamSlug: String!, $sequence: Int!, $clusterSlug: String!) {
    deployKotsVersion(upstreamSlug: $upstreamSlug, sequence: $sequence, clusterSlug: $clusterSlug)
  }
`;

export const deployKotsVersion = gql(deployKotsVersionRaw);

export const updateRegistryDetailsRaw = `
  mutation updateRegistryDetails($registryDetails: AppRegistryDetails!) {
    updateRegistryDetails(registryDetails: $registryDetails)
  }
`;

export const updateRegistryDetails = gql(updateRegistryDetailsRaw);


export const resumeInstallOnlineRaw = `
  mutation resumeInstallOnline($slug: String!) {
    resumeInstallOnline(slug: $slug) {
      hasPreflight
      slug
      isConfigurable
    }
  }
`;

export const resumeInstallOnline = gql(resumeInstallOnlineRaw);

export const updateAppConfig = gql`
  mutation updateAppConfig($slug: String!, $sequence: Int!, $configGroups: [KotsConfigGroupInput]!, $createNewVersion: Boolean) {
    updateAppConfig(slug: $slug, sequence: $sequence, configGroups: $configGroups, createNewVersion: $createNewVersion)
  }
`;

export const updateDownstreamsStatus = gql`
  mutation updateDownstreamsStatus($slug: String!, $sequence: Int!, $status: String!) {
    updateDownstreamsStatus(slug: $slug, sequence: $sequence, status: $status)
  }
`;

export const updateKotsApp = gql`
  mutation updateKotsApp($appId: String!, $appName: String, $iconUri: String) {
    updateKotsApp(appId: $appId, appName: $appName, iconUri: $iconUri)
  }
`;

export const createGitOpsRepoRaw = `
  mutation createGitOpsRepo($gitOpsInput: KotsGitOpsInput!) {
    createGitOpsRepo(gitOpsInput: $gitOpsInput)
  }
`;
export const createGitOpsRepo = gql(createGitOpsRepoRaw);

export const updateGitOpsRepoRaw = `
  mutation updateGitOpsRepo($gitOpsInput: KotsGitOpsInput!, $uriToUpdate: String) {
    updateGitOpsRepo(gitOpsInput: $gitOpsInput, uriToUpdate: $uriToUpdate)
  }
`;
export const updateGitOpsRepo = gql(updateGitOpsRepoRaw);

export const updateAppGitOpsRaw = `
  mutation updateAppGitOps($appId: String!, $clusterId: String!, $gitOpsInput: KotsGitOpsInput!) {
    updateAppGitOps(appId: $appId, clusterId: $clusterId, gitOpsInput: $gitOpsInput)
  }
`;
export const updateAppGitOps = gql(updateAppGitOpsRaw);

export const resetGitOpsDataRaw = `
  mutation resetGitOpsData {
    resetGitOpsData
  }
`;
export const resetGitOpsData = gql(resetGitOpsDataRaw);

export const setPrometheusAddress = gql`
  mutation setPrometheusAddress($value: String!) {
    setPrometheusAddress(value: $value)
  }
`;

export const syncAppLicense = gql`
  mutation syncAppLicense($appSlug: String!, $airgapLicense: String) {
    syncAppLicense(appSlug: $appSlug, airgapLicense: $airgapLicense) {
      id
      expiresAt
      channelName
      licenseSequence
      licenseType
      entitlements {
        title
        value
        label
      }
    }
  }
`;

export const testGitOpsConnectionRaw = `
  mutation testGitOpsConnection($appId: String!, $clusterId: String!) {
    testGitOpsConnection(appId: $appId, clusterId: $clusterId)
  }
`;
export const testGitOpsConnection = gql(testGitOpsConnectionRaw);

export const disableAppGitopsRaw = `
  mutation disableAppGitops($appId: String!, $clusterId: String!) {
    disableAppGitops(appId: $appId, clusterId: $clusterId)
  }
`;
export const disableAppGitops = gql(disableAppGitopsRaw);

export const ignorePreflightPermissionErrorsRaw = `
mutation ignorePreflightPermissionErrors($appSlug: String, $clusterSlug: String, $sequence: Int) {
  ignorePreflightPermissionErrors(appSlug: $appSlug, clusterSlug: $clusterSlug, sequence: $sequence)
}
`;
export const ignorePreflightPermissionErrors = gql(ignorePreflightPermissionErrorsRaw);

export const retryPreflightsRaw = `
mutation retryPreflights($appSlug: String, $clusterSlug: String, $sequence: Int) {
  retryPreflights(appSlug: $appSlug, clusterSlug: $clusterSlug, sequence: $sequence)
}
`;
export const retryPreflights = gql(retryPreflightsRaw);
