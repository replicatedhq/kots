import { Phase } from "./velero";

export const snapshotTriggerKey = "kots.io/snapshot-trigger";
export const kotsAppSlugKey = "kots.io/app-slug";
export const kotsAppIdKey = "kots.io/app-id";
export const kotsClusterIdKey = "kots.io/cluster-id";
export const kotsadmLabelKey = "app.kubernetes.io/name"; // must match kots and kurl
export const kotsAppSequenceKey = "kots.io/app-sequence";
export const snapshotVolumeCountKey = "kots.io/snapshot-volume-count";
export const snapshotVolumeSuccessCountKey = "kots.io/snapshot-volume-success-count";
export const snapshotVolumeBytesKey = "kots.io/snapshot-volume-bytes";

export enum SnapshotTrigger {
  Manual = "manual",
  Schedule = "schedule",
  PreUpgrade = "pre_upgrade",
}

export interface Snapshot {
  name: string;
  status: Phase;
  trigger: SnapshotTrigger|undefined;
  appSlug: string|undefined;
  appVersion: string|undefined;
  started?: string;
  finished?: string;
  expires?: string;
  volumeCount: number;
  volumeSuccessCount: number;
  volumeBytes: number;
  volumeSizeHuman: string;
}

export interface SnapshotDetail extends Snapshot {
  namespaces: string[];
  hooks: SnapshotHook[];
  volumes: SnapshotVolume[];
  errors: SnapshotError[];
  warnings: SnapshotError[];
}

export interface SnapshotError {
  title?: string;
  message: string;
  namespace?: string;
}

export interface SnapshotVolume {
  name: string;
  sizeBytesHuman?: string;
  doneBytesHuman?: string;
  completionPercent?: number;
  timeRemainingSeconds?: number;
  started?: string;
  finished?: string;
  phase?: Phase;
}

export enum SnapshotHookPhase {
  Pre = "pre",
  Post = "post",
}

export interface SnapshotHook {
  namespace: string;
  phase: SnapshotHookPhase,
  podName: string;
  containerName: string;
  command: string;
  hookName: string;
  stdout: string;
  stderr: string;
  started: string;
  finished: string;
  errors: SnapshotError[],
  warnings: SnapshotError[],
}

export interface RestoreDetail {
  name: string;
  phase: Phase,
  volumes: RestoreVolume[];
  errors: SnapshotError[];
  warnings: SnapshotError[];
  active: boolean;
}

export interface RestoreVolume {
  name: string;
  phase: Phase,
  podName: string;
  podNamespace: string;
  podVolumeName: string;
  sizeBytesHuman?: string;
  doneBytesHuman?: string;
  completionPercent?: number;
  timeRemainingSeconds?: number;
  started? : string;
  finished?: string;
}

export interface ScheduledSnapshot {
  id: string;
  appId: string;
  scheduledTimestamp: string;
  // name of Backup CR will be set once scheduled
  backupName?: string;
}
