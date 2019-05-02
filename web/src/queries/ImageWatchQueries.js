import gql from "graphql-tag";

export const getImageWatch = gql`
 query imageWatchItems($batchId: String!) {
   imageWatchItems(batchId: $batchId) {
    id,
    name,
    lastCheckedOn,
    isPrivate,
    versionDetected,
    latestVersion,
    compatibleVersion,
    versionsBehind,
    path
   }
 }
`;
