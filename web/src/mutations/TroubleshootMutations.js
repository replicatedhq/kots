import gql from "graphql-tag";

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
