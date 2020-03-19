export interface SnapshotConfig {
  autoEnabled: boolean;
  autoSchedule: SnapshotSchedule | null;
  ttl: SnapshotTTl;
}

export interface SnapshotSettings {
  store: SnapshotStore|null;
}

export interface SnapshotSchedule {
  schedule: string;
}

export interface SnapshotTTl {
  inputValue: string;
  inputTimeUnit: string;
  converted: string;
}

export enum SnapshotProvider {
  S3AWS = "aws",
  S3Compatible = "s3compatible",
  Azure = "azure",
  Google = "gcp",
}

export interface SnapshotStoreS3AWS {
  region: string;
  accessKeyID?: string;
  accessKeySecret?: string;
}

export interface SnapshotStoreS3Compatible extends SnapshotStoreS3AWS {
  endpoint: string;
}

export enum AzureCloudName {
  Public = "AzurePublicCloud",
  USGovernment = "AzureUSGovernmentCloud",
  China = "AzureChinaCloud",
  German = "AzureGermanCloud",
}

export interface SnapshotStoreAzure {
  resourceGroup: string;
  storageAccount: string;
  subscriptionID: string;
  tenantID: string;
  clientID: string;
  clientSecret: string;
  cloudName: AzureCloudName;
}

export interface SnapshotStoreGoogle {
  serviceAccount: string;
}

export interface SnapshotStore {
  provider: SnapshotProvider;
  bucket: string;
  path?: string;
  s3AWS?: SnapshotStoreS3AWS;
  s3Compatible?: SnapshotStoreS3Compatible;
  azure?: SnapshotStoreAzure;
  google?: SnapshotStoreGoogle;
}
