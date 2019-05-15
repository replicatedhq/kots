import gql from "graphql-tag";

export const getImageWatchRaw = `
  query imageWatches($batchId: String!) {
    imageWatches(batchId: $batchId) {
      id
      name
      lastCheckedOn
      isPrivate
      versionDetected
      latestVersion
      compatibleVersion
      versionsBehind
      path
    }
  }
`;
export const getImageWatch = gql(getImageWatchRaw);
