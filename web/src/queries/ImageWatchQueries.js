import gql from "graphql-tag";

export const getImageWatchRaw = `
  query imageWatchItems($batchId: String!) {
    imageWatchItems(batchId: $batchId) {
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
