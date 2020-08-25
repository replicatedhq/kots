import gql from "graphql-tag";

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


export const saveSnapshotConfigRaw = `	
  mutation saveSnapshotConfig($appId: String!, $inputValue: Int!, $inputTimeUnit: String!, $schedule: String!, $autoEnabled: Boolean!) {	
    saveSnapshotConfig(appId: $appId, inputValue: $inputValue, inputTimeUnit: $inputTimeUnit, schedule: $schedule, autoEnabled: $autoEnabled)	
  }	
`;	
export const saveSnapshotConfig = gql(saveSnapshotConfigRaw);