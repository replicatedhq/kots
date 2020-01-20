import {
  V1ObjectMeta,
  V1LabelSelector,
} from "@kubernetes/client-node";

export interface Backup {
  apiVersion: string,
  kind: string,
  metadata: V1ObjectMeta,
  spec: BackupSpec,
  status?: BackupStatus,
}

export interface BackupSpec {
  excludedNamespaces?: Array<string>,
  excludedResources?: Array<string>,
  hooks?: Hooks,
  includeClusterResources?: boolean|null,
  includedNamespaces?: Array<string>,
  includedResources?: Array<string>,
  labelSelector?: V1LabelSelector,
  snapshotVolumes?: boolean|null,
  storageLocation: string,
  ttl: string,
  volumeSnapshotLocations?: Array<string>,
}

export interface Hooks {
  resources: Array<ResourceHook>
}

export interface ResourceHook {
  name: string,
  includedNamespaces: Array<string>,
  excludedNamespaces: Array<string>,
  includedResources: Array<string>,
  excludedResources: Array<string>,
  labelSelector: V1LabelSelector,
  pre: Array<Hook>,
  post: Array<Hook>,
}

export interface Hook {
  exec: Exec,
}

export interface Exec {
  container?: string,
  command: Array<string>,
  onError: "Fail"|"Continue",
  timeout?: string,
}

export interface BackupStatus {
  completionTimestamp: string,
  errors: number,
  expiration: string,
  phase: Phase,
  startTimestamp: string,
  validationErrors: Array<any>,
  version: number,
  volumeSnapshotsAttempted: number,
  volumeSnapshotsCompleted: number,
  warnings: number,
}

export enum Phase {
  New = "New",
  FailedValidation = "FailedValidation",
  InProgress = "InProgress",
  Completed = "Completed",
  PartiallyFailed = "PartiallyFailed",
  Failed = "Failed",
}

export interface Restore {
  apiVersion: string,
  kind: string,
  metdata: V1ObjectMeta,
  spec: {
    backupName: string,
  },
  status?: RestoreStatus,
}

export interface RestoreStatus {
  phase: Phase,
  warnings?: number,
  errors?: number,
  failureReason?: string,
  validationErrors?: Array<string>,
}
