import gql from "graphql-tag";

export const manualSnapshotRaw = `
  mutation manualSnapshot($appId: String!) {
    manualSnapshot(appId: $appId)
  }
`;
export const manualSnapshot = gql(manualSnapshotRaw);

export const deleteSnapshot = gql`
  mutation deleteSnapshot($snapshotName: String!) {
    deleteSnapshot(snapshotName: $snapshotName)
  }
`;

export const snapshotProviderAWSRaw = `
  mutation snapshotProviderAWS($bucket: String!, $prefix: String, $region: String!, $accessKeyID: String, $accessKeySecret: String) {
    snapshotProviderAWS(bucket: $bucket, prefix: $prefix, region: $region, accessKeyID: $accessKeyID, accessKeySecret: $accessKeySecret)
  }
`;
export const snapshotProviderAWS = gql(snapshotProviderAWSRaw);

export const snapshotProviderS3CompatibleRaw = `
  mutation snapshotProviderS3Compatible($bucket: String!, $prefix: String, $region: String!, $endpoint: String!, $accessKeyID: String, $accessKeySecret: String) {
    snapshotProviderS3Compatible(bucket: $bucket, prefix: $prefix, region: $region, endpoint: $endpoint, accessKeyID: $accessKeyID, accessKeySecret: $accessKeySecret)
  }
`;
export const snapshotProviderS3Compatible = gql(snapshotProviderS3CompatibleRaw);

export const snapshotProviderAzureRaw = `
  mutation snapshotProviderAzure($bucket: String!, $prefix: String, $tenantID: String!, $resourceGroup: String!, $storageAccount: String!, $subscriptionID: String!, $clientID: String!, $clientSecret: String!, $cloudName: String!) {
    snapshotProviderAzure(bucket: $bucket, prefix: $prefix, tenantID: $tenantID, resourceGroup: $resourceGroup, storageAccount: $storageAccount, subscriptionID: $subscriptionID, clientID: $clientID, clientSecret: $clientSecret, cloudName: $cloudName)
  }
`;
export const snapshotProviderAzure = gql(snapshotProviderAzureRaw);

export const snapshotProviderGoogleRaw = `
  mutation snapshotProviderGoogle($bucket: String!, $prefix: String, $serviceAccount: String!) {
    snapshotProviderGoogle(bucket: $bucket, prefix: $prefix, serviceAccount: $serviceAccount)
  }
`;
export const snapshotProviderGoogle = gql(snapshotProviderGoogleRaw);

export const saveSnapshotConfigRaw = `
  mutation saveSnapshotConfig($appId: String!, $inputValue: Int!, $inputTimeUnit: String!, $userSelected: String!, $schedule: String!, $autoEnabled: Boolean!) {
    saveSnapshotConfig(appId: $appId, inputValue: $inputValue, inputTimeUnit: $inputTimeUnit, userSelected: $userSelected, schedule: $schedule, autoEnabled: $autoEnabled)
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