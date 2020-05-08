import gql from "graphql-tag";

export const archiveSupportBundle = gql`
  mutation archiveSupportBundle($id: String!) {
    archiveSupportBundle(id: $id)
  }
`;

export const collectSupportBundle = gql`
  mutation collectSupportBundle($appId: String, $clusterId: String) {
    collectSupportBundle(appId: $appId, clusterId: $clusterId)
  }
`;
