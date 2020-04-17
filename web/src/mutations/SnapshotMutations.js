import gql from "graphql-tag";

export const deleteSnapshot = gql`
  mutation deleteSnapshot($snapshotName: String!) {
    deleteSnapshot(snapshotName: $snapshotName)
  }
`;

export const saveSnapshotConfig = gql(saveSnapshotConfigRaw);

export const restoreSnapshotRaw = `
  mutation restoreSnapshot($snapshotName: String!) {
    restoreSnapshot(snapshotName: $snapshotName) {
      name
    }
  }
`;
export const restoreSnapshot = gql(restoreSnapshotRaw);

export const cancelRestoreRaw = `
  mutation cancelRestore($appId: String!) {
    cancelRestore(appId: $appId)
  }
`;
export const cancelRestore = gql(cancelRestoreRaw);
