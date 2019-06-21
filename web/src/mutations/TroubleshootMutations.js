import gql from "graphql-tag";

export const uploadSupportBundle = gql`
  mutation uploadSupportBundle($appId: String, $customerId: String, $size: Int) {
    uploadSupportBundle(appId: $appId, customerId: $customerId, size: $size) {
      uploadUri,
      supportBundle {
        id,
        size,
        status,
        createdAt,
        uri,
        signedUri
      }
    }
  }
`;

export const markSupportBundleUploaded = gql`
  mutation markSupportBundleUploaded($id: String!) {
    markSupportBundleUploaded(id: $id) {
      id
      slug
    }
  }
`;

export const updateSupportBundle = gql`
  mutation updateSupportBundle($id: String!, $name: String, $shareTeamIDs: [String]) {
    updateSupportBundle(id: $id, name: $name, shareTeamIDs: $shareTeamIDs) {
      id
    }
  }
`;

export const archiveSupportBundle = gql`
  mutation archiveSupportBundle($id: String!) {
    archiveSupportBundle(id: $id)
  }
`;
