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
    }
  }
`
export const uploadKotsLicense = gql(uploadKotsLicenseRaw);

export const getAirgapPutUrlRaw = `
  mutation getAirgapPutUrl($filename: String!) {
    getAirgapPutUrl(filename: $filename)
  }
`
export const getAirgapPutUrl = gql(getAirgapPutUrlRaw);

export const markAirgapBundleUploadedRaw = `
  mutation markAirgapBundleUploaded($filename: String!) {
    markAirgapBundleUploaded(filename: $filename)
  }
`
export const markAirgapBundleUploaded = gql(markAirgapBundleUploadedRaw);

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
