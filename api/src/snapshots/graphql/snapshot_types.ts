const SnapshotConfig = `
  type SnapshotConfig {
    autoEnabled: Boolean
    autoSchedule: SnapshotSchedule
    ttl: SnapshotTTl
    store: SnapshotStore
  }
`;

const SnapshotSchedule = `
  type SnapshotSchedule {
    schedule: String
  }
`;

const SnapshotTTl = `
  type SnapshotTTl {
    inputValue: String
    inputTimeUnit: String
    converted: String
  }
`;

const SnapshotStore = `
  type SnapshotStore {
    provider: String
    bucket: String
    path: String
    s3AWS: SnapshotStoreS3AWS
    azure: SnapshotStoreAzure
    s3Compatible: SnapshotStoreS3Compatible
    google: SnapshotStoreGoogle
  }
`;

const SnapshotStoreS3AWS = `
  type SnapshotStoreS3AWS {
    region: String
    accessKeyID: String
    accessKeySecret: String
  }
`;


const SnapshotStoreS3Compatible = `
  type SnapshotStoreS3Compatible {
    endpoint: String
    region: String
    accessKeyID: String
    accessKeySecret: String
  }
`;

const SnapshotStoreAzure = `
  type SnapshotStoreAzure {
    tenantID: String
    resourceGroup: String
    storageAccount: String
    subscriptionID: String
    clientID: String
    clientSecret: String
    cloudName: String
  }
`;

const SnapshotStoreGoogle = `
  type SnapshotStoreGoogle {
    serviceAccount: String
  }
`;

const Snapshot = `
  type Snapshot {
    name: String
    status: String
    trigger: String
    appVersion: String
    started: String
    finished: String
    expires: String
    volumeCount: Int
    volumeSuccessCount: Int
    volumeSizeHuman: String
  }
`

const SnapshotDetail = `
  type SnapshotDetail {
    name: String
    status: String
    volumeSizeHuman: String
    namespaces: [String]
    hooks: [SnapshotHook]
    volumes: [SnapshotVolume]
    errors: [SnapshotError]
    warnings: [SnapshotError]
  }
`

const SnapshotError = `
  type SnapshotError {
    title: String
    message: String
    namespace: String
  }
`;

const SnapshotVolume = `
  type SnapshotVolume {
    name: String
    sizeBytesHuman: String
    doneBytesHuman: String
    completionPercent: Int
    timeRemainingSeconds: Int
    started: String
    finished: String
    phase: String
  }
`;

const SnapshotHook = `
  type SnapshotHook {
    hookName: String
    namespace: String
    phase: String
    podName: String
    command: String
    containerName: String
    stdout: String
    stderr: String
    started: String
    finished: String
    errors: [SnapshotError]
    warnings: [SnapshotError]
  }
`;

const RestoreDetail = `
  type RestoreDetail {
    name: String
    phase: String
    volumes: [RestoreVolume]
    errors: [SnapshotError]
    warnings: [SnapshotError]
    active: Boolean
  }
`;

const RestoreVolume = `
  type RestoreVolume {
    name: String
    phase: String
    podName: String
    podNamespace: String
    podVolumeName: String
    sizeBytesHuman: String
    doneBytesHuman: String
    completionPercent: Int
    timeRemainingSeconds: Int
    started: String
    finished: String
  }
`;

export default [
  SnapshotConfig,
  SnapshotStore,
  SnapshotSchedule,
  SnapshotTTl,
  SnapshotStoreS3AWS,
  SnapshotStoreS3Compatible,
  SnapshotStoreAzure,
  SnapshotStoreGoogle,
  Snapshot,
  SnapshotDetail,
  SnapshotError,
  SnapshotVolume,
  SnapshotHook,
  RestoreDetail,
  RestoreVolume
]
