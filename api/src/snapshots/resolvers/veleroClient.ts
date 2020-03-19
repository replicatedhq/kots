import crypto from "crypto";
import zlib from "zlib";
import querystring from "querystring";
import { join } from "path";
import moment from "moment";
import * as _ from "lodash";
import prettyBytes from "pretty-bytes";
import {
  CoreV1Api,
  KubeConfig,
  V1Secret,
} from "@kubernetes/client-node";
import { ReplicatedError } from "../../server/errors";
import { logger } from "../../server/logger";
import request, { RequestPromiseOptions } from "request-promise";
import {
  kotsAppSlugKey,
  kotsAppSequenceKey,
  kotsAppIdKey,
  snapshotTriggerKey,
  snapshotVolumeCountKey,
  snapshotVolumeSuccessCountKey,
  snapshotVolumeBytesKey,
  RestoreVolume,
  Snapshot,
  SnapshotDetail,
  SnapshotError,
  SnapshotHook,
  SnapshotHookPhase,
  SnapshotTrigger,
  SnapshotVolume } from "../snapshot";
import {
  SnapshotProvider,
  SnapshotStore,
  SnapshotStoreAzure,
  SnapshotStoreS3AWS,
  SnapshotStoreGoogle } from "../";
import { Backup, Phase, Restore } from "../velero";
import { base64Decode, sleep } from "../../util/utilities";
import { parseBackupLogs, ParsedBackupLogs } from "./parseBackupLogs";
import { AzureCloudName } from "../snapshot_config";

export const backupStorageLocationName = "kotsadm-velero-backend";
const awsSecretName = "aws-credentials";
const googleSecretName = "google-credentials";
const azureSecretName = "azure-credentials";
const redacted = "--- REDACTED ---";

interface VolumeSummary {
  count: number,
  success: number,
  bytes: number,
}

export class VeleroClient {
  private readonly kc: KubeConfig;
  private readonly server: string;

  constructor(
    private readonly ns: string,
  ) {
    this.kc = new KubeConfig();
    this.kc.loadFromDefault();

    const cluster = this.kc.getCurrentCluster();
    if (!cluster) {
      throw new Error("No cluster available from kubeconfig");
    }
    this.server = cluster.server;
  }

  // tslint:disable-next-line cyclomatic-complexity
  async request(method: string, path: string, body?: any): Promise<any> {
    const url = `${this.server}/apis/velero.io/v1/namespaces/${this.ns}/${path}`;
    const req = { url };
    await this.kc.applyToRequest(req);
    const options: RequestPromiseOptions = {
      method,
      body,
      simple: false,
      resolveWithFullResponse: true,
      json: true,
    };
    Object.assign(options, req);

    const response = await this.unhandledRequest(method, path, body);
    switch (response.statusCode) {
    case 200: // fallthrough
    case 201: // fallthrough
    case 204:
      return response.body
    case 400:
      logger.warn(response.body);
      break
    case 403:
      throw new ReplicatedError(`Permission denied: RBAC may be misconfigured for ${method} velero.io/v1 ${path} in namespace ${this.ns}`);
    case 404:
      throw new ReplicatedError("Not found: a requested resource may not exist or Velero may not be installed in this cluster");
    case 422:
      logger.warn(response.body);
      break;
    default:
    }

    if (response.body && response.body.message) {
      throw new ReplicatedError(response.body.message);
    }

    throw new Error(response.statusCode);
  }

  async unhandledRequest(method: string, path: string, body?: any): Promise<any> {
    const url = `${this.server}/apis/velero.io/v1/namespaces/${this.ns}/${path}`;
    const req = { url };
    await this.kc.applyToRequest(req);
    const options: RequestPromiseOptions = {
      method,
      body,
      simple: false,
      resolveWithFullResponse: true,
      json: true,
    };
    Object.assign(options, req);

    const response = await request(url, options);
    return response;
  }

  async listSnapshots(slug: string): Promise<Snapshot[]> {
    const q  = {
      labelSelector: `${kotsAppSlugKey}=${slug}`,
    };
    const body = await this.request("GET", `backups?${querystring.stringify(q)}`);
    const snapshots: Snapshot[] = [];

    for (const backup of body.items) {
      const snapshot = await this.snapshotFromBackup(backup);
      snapshots.push(snapshot);
    }

    return snapshots;
  }

  async hasUnfinishedBackup(appId: string): Promise<boolean> {
    if (!appId) {
      return false;
    }
    const body = await this.request("GET", "backups");

    return _.some(body.items, (backup: Backup) => {
      const backupAppId = backup.metadata.annotations && backup.metadata.annotations[kotsAppIdKey];
      const phase = backup.status && backup.status.phase;
      if (!phase) {
        // no status means phase new
        return true;
      }
      const isPhaseUnfinished = phase === Phase.New || phase === Phase.InProgress;

      return appId === backupAppId && isPhaseUnfinished;
    });
  }

  // tslint:disable-next-line cyclomatic-complexity
  async snapshotFromBackup(backup: Backup): Promise<Snapshot> {
    let trigger: SnapshotTrigger|undefined;

    switch (backup.metadata.annotations && backup.metadata.annotations[snapshotTriggerKey]) {
    case SnapshotTrigger.Manual:
      trigger = SnapshotTrigger.Manual;
      break;
    case SnapshotTrigger.PreUpgrade:
      trigger = SnapshotTrigger.PreUpgrade;
      break;
    case SnapshotTrigger.Schedule:
      trigger = SnapshotTrigger.Schedule;
      break;
    default:
    }

    const status = backup.status ? backup.status.phase : Phase.New;

    let volumeCount = maybeParseInt(backup.metadata.annotations && backup.metadata.annotations[snapshotVolumeCountKey]);
    let volumeSuccessCount = maybeParseInt(backup.metadata.annotations && backup.metadata.annotations[snapshotVolumeSuccessCountKey]);
    let volumeBytes = maybeParseInt(backup.metadata.annotations && backup.metadata.annotations[snapshotVolumeBytesKey]);

    if (_.isUndefined(volumeCount) || _.isUndefined(volumeSuccessCount) || _.isUndefined(volumeBytes)) {
      const { count, success, bytes } = await this.getSnapshotVolumeSummary(backup.metadata.name!);
      volumeCount = count;
      volumeSuccessCount = success;
      volumeBytes = bytes;

      // save computed summary as annotations if snapshot is finished
      if (status !== Phase.New && status !== Phase.InProgress) {
        backup.metadata.annotations = backup.metadata.annotations || {};
        backup.metadata.annotations[snapshotVolumeCountKey] = volumeCount.toString();
        backup.metadata.annotations[snapshotVolumeSuccessCountKey] = volumeSuccessCount.toString();
        backup.metadata.annotations[snapshotVolumeBytesKey] = volumeBytes.toString();
        await this.request("PUT", `backups/${backup.metadata.name}`, backup);
      }
    }

    return {
      name: backup.metadata.name!,
      status,
      trigger,
      appSlug: backup.metadata.annotations && backup.metadata.annotations[kotsAppSlugKey],
      appVersion: backup.metadata.annotations && backup.metadata.annotations[kotsAppSequenceKey],
      started: backup.status && backup.status.startTimestamp,
      finished: backup.status && backup.status.completionTimestamp,
      expires: backup.status && backup.status.expiration,
      volumeCount,
      volumeSuccessCount,
      volumeSizeHuman: prettyBytes(volumeBytes),
      volumeBytes,
    };
  }

  async getSnapshotVolumeSummary(backupName: string): Promise<VolumeSummary> {
    let count = 0;
    let success = 0;
    let bytes = 0;

    const q  = {
      labelSelector: `velero.io/backup-name=${getValidName(backupName)}`,
    };
    const body = await this.request("GET", `podvolumebackups?${querystring.stringify(q)}`);

    _.each(body.items, (pvb) => {
      count++;
      if (pvb.status.phase === "Completed") {
        success++;
      }
      if (_.isNumber(pvb.status.progress.bytesDone)) {
        bytes += pvb.status.progress.bytesDone;
      }
    });

    return { count, success, bytes };
  }

  async createBackup(backup: Backup): Promise<Backup> {
    const body = await this.request("POST", "backups", backup);

    return body;
  }

  async readBackup(name: string): Promise<Backup> {
    return this.request("GET", `backups/${name}`);
  }

  async readRestore(name: string): Promise<Restore|null> {
    const response = await this.unhandledRequest("GET", `restores/${name}`);
    if (response.statusCode === 200) {
      return response.body;
    }
    if (response.statusCode === 404) {
      return null;
    }
    if (response.statusCode === 403) {
      throw new ReplicatedError(`Permission denied reading Restore ${name} from namespace ${this.ns}`);
    }
    if (response.body && response.body.message) {
      throw new ReplicatedError(response.body.message);
    }

    throw new Error(`Read Restore ${name} from namespace ${this.ns}: ${response.statusCode}`);
  }

  async listRestoreVolumes(name: string): Promise<RestoreVolume[]> {
    const q = {
      labelSelector: `velero.io/restore-name=${getValidName(name)}`,
    };
    const volumeList = await this.request("GET", `podvolumerestores?${querystring.stringify(q)}`);
    const volumes: RestoreVolume[] = [];

    _.each(volumeList.items, (pvr) => {
      const rv: RestoreVolume = {
        name: pvr.metadata.name,
        phase: Phase.New,
        podName: pvr.spec.pod.name,
        podNamespace: pvr.spec.pod.namespace,
        podVolumeName: pvr.spec.volume,
      };

      if (pvr.status) {
        rv.started = pvr.status.startTimestamp;
        if (pvr.status.completionTimestamp) {
          rv.finished = pvr.status.completionTimestamp;
        }
        if (pvr.status.phase) {
          rv.phase = pvr.status.phase;
        }
        if (pvr.status.progress) {
          rv.sizeBytesHuman = pvr.status.progress.totalBytes ? prettyBytes(pvr.status.progress.totalBytes) : "0 B";
          rv.doneBytesHuman = pvr.status.progress.bytesDone ? prettyBytes(pvr.status.progress.bytesDone) : "0 B";
          rv.completionPercent = Math.round(pvr.status.progress.bytesDone / pvr.status.progress.totalBytes * 100);
          const bytesPerSecond = pvr.status.progress.bytesDone / moment().diff(moment(pvr.status.startTimestamp), "seconds");
          const bytesRemaining = pvr.status.progress.totalBytes - pvr.status.progress.bytesDone;
          rv.timeRemainingSeconds = Math.round(bytesRemaining / bytesPerSecond);
        }
      }

      volumes.push(rv);
    });

    return volumes;
  }

  // tslint:disable-next-line cyclomatic-complexity
  async getSnapshotDetail(name: string): Promise<SnapshotDetail> {
    const path = `backups/${name}`;
    const backup = await this.request("GET", path);
    const snapshot = await this.snapshotFromBackup(backup);

    const q = {
      labelSelector: `velero.io/backup-name=${getValidName(name)}`,
    };
    const volumeList = await this.request("GET", `podvolumebackups?${querystring.stringify(q)}`);
    const volumes: SnapshotVolume[] = [];

    _.each(volumeList.items, (pvb) => {
      const sv: SnapshotVolume = {
        name: pvb.metadata.name,
      };
      if (pvb.status) {
        sv.started = pvb.status.startTimestamp;
        sv.finished = pvb.status.completionTimestamp;
        sv.phase = pvb.status.phase;
        if (pvb.status.progress) {
          // progress object is empty if volume size was 0
          sv.sizeBytesHuman = pvb.status.progress.totalBytes ? prettyBytes(pvb.status.progress.totalBytes) : "0 B";
          sv.doneBytesHuman = pvb.status.progress.bytesDone ? prettyBytes(pvb.status.progress.bytesDone): "0 B";
          sv.completionPercent = Math.round(pvb.status.progress.bytesDone / pvb.status.progress.totalBytes * 100);
          const bytesPerSecond = pvb.status.progress.bytesDone / moment().diff(moment(pvb.status.startTimestamp), "seconds");
          const bytesRemaining = pvb.status.progress.totalBytes - pvb.status.progress.bytesDone;
          sv.timeRemainingSeconds = Math.round(bytesRemaining / bytesPerSecond);
        }
      }
      volumes.push(sv);
    });

    let logs;
    if (snapshot.status === Phase.Completed || snapshot.status === Phase.PartiallyFailed || snapshot.status === Phase.Failed) {
      try {
        logs = await this.getBackupLogs(name);
      } catch(e) {
        logger.error(`Failed to get backup logs: ${e.message}`);
      }
    }

    const errors: SnapshotError[] = logs ? logs.errors : [];

    _.each(backup.status.validationErrors, (message: string) => {
      errors.push({
        title: "Validation Error",
        message,
      });
    });

    return {
      ...snapshot,
      namespaces: backup.spec.includedNamespaces,
      volumes,
      errors,
      hooks: logs && logs.execs,
      warnings: logs && logs.warnings,
    };
  }

  async getBackupLogs(name: string): Promise<ParsedBackupLogs> {
    const url = await this.getDownloadURL("BackupLog", name);
    const options = {
      method: "GET",
      simple: true,
      encoding: null, // get a Buffer for the response
    };
    const buffer = await request(url, options);

    return new Promise((resolve, reject) => {
      zlib.gunzip(buffer, (err, buf) => {
        if (err) {
          reject(err);
          return;
        }
        resolve(parseBackupLogs(buf));
      });
    });
  }

  async getRestoreResults(name: string): Promise<any> {
    const url = await this.getDownloadURL("RestoreResults", name);
    const options = {
      method: "GET",
      simple: true,
      encoding: null, // get a Buffer for the response
    };
    const buffer = await request(url, options);

    return new Promise((resolve, reject) => {
      zlib.gunzip(buffer, (err, buf) => {
        if (err) {
          reject(err);
          return;
        }
        resolve(JSON.parse(buf.toString()));
      });
    });
  }

  async getDownloadURL(kind, name: string): Promise<string> {
    const drname = getValidName(`${kind.toLowerCase()}-${name}-${Date.now()}`);

    const downloadrequest = {
      apiVersion: "velero.io/v1",
      kind: "DownloadRequest",
      metadata: {
        name: drname,
      },
      spec: {
        target: {
          kind,
          name,
        },
      }
    };
    await this.request("POST", "downloadrequests", downloadrequest);

    for (let i = 0; i < 30; i++) {
      const body = await this.request("GET", `downloadrequests/${drname}`);
      if (body.status && body.status.downloadURL) {
        await this.request("DELETE", `downloadrequests/${drname}`);
        return body.status.downloadURL;
      }
      await sleep(1);
    }

    throw new Error(`Timed out waiting for DownloadRequest for ${kind}/${name} logs`);
  }

  // tslint:disable-next-line cyclomatic-complexity
  async readSnapshotStore(): Promise<SnapshotStore|null> {
    const corev1 = this.kc.makeApiClient(CoreV1Api);
    const bsls = await this.request("GET", `backupstoragelocations`); 
    const bsl: any = _.find(bsls.items, (bslItem) => {
      return bslItem.metadata.name === backupStorageLocationName;
    });

    if (!bsl) {
      return null;
    }

    const store: SnapshotStore = {
      provider: bsl.spec.provider,
      bucket: bsl.spec.objectStorage.bucket,
      path: bsl.spec.objectStorage.prefix,
    };

    switch (store.provider) {
      case SnapshotProvider.S3AWS:
        const {accessKeyID, accessKeySecret} = await readAWSCredentialsSecret(corev1, this.ns);

        if (bsl.spec.config.s3Url) {
          store.provider = SnapshotProvider.S3Compatible
          store.s3Compatible = {
            region: bsl.spec.config.region,
            endpoint: bsl.spec.config.s3Url,
            accessKeyID: accessKeyID,
          };
          if (accessKeySecret) {
            store.s3Compatible.accessKeySecret = redacted;
          }
        } else {
          store.s3AWS = {
            region: bsl.spec.config.region,
            accessKeyID,
          };
          if (accessKeySecret) {
            store.s3AWS.accessKeySecret = redacted;
          }
        }
        break;

      case SnapshotProvider.Azure:
        const creds = await readAzureCredentialsSecret(corev1, this.ns);

        store.azure = {
          resourceGroup: bsl.spec.config.resourceGroup,
          storageAccount: bsl.spec.config.storageAccount,
          subscriptionID: bsl.spec.config.subscriptionId,
          tenantID: creds.tenantID || "",
          clientID: creds.clientID || "",
          clientSecret: creds.clientSecret ? redacted : "",
          cloudName: creds.cloudName || AzureCloudName.Public,
        };
        break;

      case SnapshotProvider.Google:
        const serviceAccount = await readGoogleCredentialsSecret(corev1, this.ns);

        store.google = {
          serviceAccount: serviceAccount ? redacted : "",
        };

      default:
    }

    return store;
  }

  // Create a new BackupStorageLocation for the app copied from kotsadm-velero-backend
  async maybeCreateAppBackend(slug): Promise<void> {
    const currentBSLResponse = await this.unhandledRequest("GET", `backupstoragelocations/${backupStorageLocationName}`);
    if (currentBSLResponse.statusCode !== 200) {
      return;
    }

    const bsl = {
      apiVersion: "velero.io/v1",
      kind: "BackupStorageLocation",
      metadata: {
        name: slug,
        namespace: this.ns,
      },
      spec: currentBSLResponse.body.spec,
    };
    bsl.spec.objectStorage.prefix = join(bsl.spec.objectStorage.prefix, slug);

    const postResponse = await this.unhandledRequest("POST", `backupstoragelocations/${slug}`, bsl);
    if (postResponse.statusCode !== 201 && postResponse.statusCode !== 409) {
      logger.error(postResponse.body);
      throw new ReplicatedError(`Failed to create new BackupStorageLocation for app ${slug}`);
    }
  }

  // tslint:disable-next-line
  async saveSnapshotStore(store: SnapshotStore, slugs: string[]): Promise<void> {
    let currentBSL: any;
    const currentBSLResponse = await this.unhandledRequest("GET", `backupstoragelocations/${backupStorageLocationName}`);
    if (currentBSLResponse.statusCode === 200) {
      currentBSL = currentBSLResponse.body;
    }

    const backupStorageLocation = {
      apiVersion: "velero.io/v1",
      kind: "BackupStorageLocation",
      metadata: {
        name: backupStorageLocationName,
        namespace: this.ns,
      },
      spec: {
        provider: store.provider,
        objectStorage: {
          bucket: store.bucket,
          prefix: store.path,
        },
        config: currentBSL ? currentBSL.spec.config : {},
      },
    };
    const corev1 = this.kc.makeApiClient(CoreV1Api);
    let credentialsSecret: V1Secret|null;

    switch (store.provider) {
    case SnapshotProvider.S3AWS:
      if (!_.isObject(store.s3AWS)) {
        throw new ReplicatedError("s3AWS store configuration is required");
      }
      backupStorageLocation.spec.config = {
        region: store.s3AWS!.region,
      };
      credentialsSecret = await awsCredentialsSecret(corev1, this.ns, store.s3AWS!);
      break;

    case SnapshotProvider.S3Compatible:
      if (!_.isObject(store.s3Compatible)) {
        throw new ReplicatedError("s3Compatible store configuration is required");
      }
      backupStorageLocation.spec.provider = SnapshotProvider.S3AWS;
      backupStorageLocation.spec.config.region = store.s3Compatible!.region;
      backupStorageLocation.spec.config.s3Url = store.s3Compatible!.endpoint;
      backupStorageLocation.spec.config.s3ForcePathStyle = "true";

      credentialsSecret = await awsCredentialsSecret(corev1, this.ns, store.s3Compatible!);
      break;

    case SnapshotProvider.Azure:
      if (!_.isObject(store.azure)) {
        throw new ReplicatedError("azure store configuration is required");
      }
      backupStorageLocation.spec.config = {
        resourceGroup: store.azure!.resourceGroup,
        storageAccount: store.azure!.storageAccount,
        subscriptionId: store.azure!.subscriptionID,
      };
      credentialsSecret = await azureCredentialsSecret(corev1, this.ns, store.azure!);
      break;

    case SnapshotProvider.Google:
      if (!_.isObject(store.google)) {
        throw new ReplicatedError("google store configuration is required");
      }
      backupStorageLocation.spec.config = {};
      credentialsSecret = await googleCredentialsSecret(corev1, this.ns, store.google!);
      break;

    default:
      throw new ReplicatedError(`unknown snapshot provider: ${_.escape(store.provider)}`);
    }

    if (currentBSL) {
      currentBSL.spec = backupStorageLocation.spec;
      await this.request("PUT", `backupstoragelocations/${backupStorageLocationName}`, currentBSL);
    } else {
      await this.request("POST", "backupstoragelocations", backupStorageLocation);
    }

    for (const slug of slugs) {
      let appBSL: any;
      const resourceURL = `backupstoragelocations/${slug}`;

      const appBSLResponse = await this.unhandledRequest("GET", resourceURL);
      switch (appBSLResponse.statusCode) {
      case 200:
        appBSL = appBSLResponse.body
        appBSL.spec = _.cloneDeep(backupStorageLocation.spec);
        appBSL.spec.objectStorage.prefix = store.path ? join(store.path, slug) : slug;
        await this.request("PUT", resourceURL, appBSL);
        break;
      case 404:
        appBSL = _.cloneDeep(backupStorageLocation);
        appBSL.metadata.name = slug;
        appBSL.spec.objectStorage.prefix = store.path ? join(store.path, slug) : slug;
        await this.request("POST", resourceURL, appBSL);
        break;
      default:
        logger.error(appBSLResponse.body);
        throw new ReplicatedError(`Failed to GET BackupStorageLocation ${slug}: ${appBSLResponse.statusCode}`);
      }
    }

    if (!credentialsSecret) {
      return;
    }

    // TODO use status codes to simplify
    try {
      const { response } = await corev1.readNamespacedSecret(credentialsSecret.metadata!.name!, this.ns);
      if (response.statusCode === 200) {
        try {
          await corev1.replaceNamespacedSecret(credentialsSecret.metadata!.name!, this.ns, credentialsSecret);
        } catch(e) {
          logger.error(e);
          return;
        }
      } else {
        try {
          await corev1.createNamespacedSecret(this.ns, credentialsSecret);
        } catch(e) {
          logger.error(e);
          return;
        }
      }
    } catch(e) {
      try {
        await corev1.createNamespacedSecret(this.ns, credentialsSecret);
      } catch(e) {
        logger.error(e);
        return;
      }
    }
  }

  async listBackends(): Promise<string[]> {
    const body = await this.request("GET", "backupstoragelocations");

    return _.map(body.items, (item: any) => {
      return item.metadata.name;
    });
  }

  async deleteSnapshot(backupName: string): Promise<void> {
    const dbr = {
      apiVersion: "velero.io/v1",
      kind: "DeleteBackupRequest",
      metadata: {
        name: `${backupName}-${Date.now()}`,
        namespace: this.ns,
      },
      spec: { backupName },
    };
    await this.request("POST", "deletebackuprequests", dbr);
  }

  async restore(backupName: string, restoreName: string): Promise<void> {
    const restore = {
      apiVersion: "velero.io/v1",
      kind: "Restore",
      metadata: {
        name: restoreName,
        namespace: "velero",
      },
      spec: {
        backupName,
      },
    };

    await this.request("POST", "restores", restore);
  }

  async isVeleroInstalled(): Promise<boolean> {
    const url = `${this.server}/apis/velero.io/`;
    const req = { url };
    await this.kc.applyToRequest(req);
    const options: RequestPromiseOptions = {
      method: "GET",
      simple: false,
      resolveWithFullResponse: true,
      json: true,
    };
    Object.assign(options, req);

    const response = await request(url, options);
    if (response.statusCode === 200) {
      return true;
    }
    if (response.statusCode === 404) {
      return false;
    }

    throw new Error(`GET ${url}: ${response.statusCode}`);
  }
}

function maybeParseInt(s: string|undefined): number|undefined {
  if (_.isString(s)) {
    const i = parseInt(s, 10)
    if (_.isNumber(i)) {
      return i;
    }
    return;
  }
}

// https://github.com/vmware-tanzu/velero/blob/be140985c5232710c9ed5ff6f85d630b96b9b7be/pkg/label/label.go#L31
function getValidName(label: string): string {
  const DNS1035LabelMaxLength = 63;

  if (label.length <= DNS1035LabelMaxLength) {
    return label;
  }
  const shasum = crypto.createHash("sha256");
  shasum.update(label);
  const sha = shasum.digest("hex");

  return label.slice(0, DNS1035LabelMaxLength - 6) + sha.slice(0, 6);
}

async function azureCredentialsSecret(corev1: CoreV1Api, namespace: string, azure: SnapshotStoreAzure): Promise<V1Secret> {
  let clientSecret = azure.clientSecret;
  if (clientSecret === redacted) {
    const creds = await readAzureCredentialsSecret(corev1, namespace);
    clientSecret = creds.clientSecret || "";
  }

  return {
    apiVersion: "v1",
    kind: "Secret",
    metadata: {
      name: "azure-credentials",
    },
    stringData: {
      cloud: `AZURE_SUBSCRIPTION_ID=${azure.subscriptionID}
AZURE_TENANT_ID=${azure.tenantID}
AZURE_CLIENT_ID=${azure.clientID}
AZURE_CLIENT_SECRET=${clientSecret}
AZURE_RESOURCE_GROUP=${azure.resourceGroup}
AZURE_CLOUD_NAME=${azure.cloudName}`,
    },
  };
}

interface AzureCreds {
  tenantID?: string,
  clientID?: string,
  clientSecret?: string,
  resourceGroup?: string,
  cloudName?: AzureCloudName,
}
// tslint:disable-next-line cyclomatic-complexity
async function readAzureCredentialsSecret(corev1: CoreV1Api, namespace: string): Promise<AzureCreds> {
  try {
    const creds: AzureCreds = {};

    let secret: any;
    try {
      const { body } = await corev1.readNamespacedSecret(azureSecretName, namespace);
      secret = body;
    } catch(e) {
      if (e.response && e.response.statusCode === 404) {
        return creds;
      }
      throw e;
    }
    if (!secret.data!.cloud) {
      return creds;
    }
    const cloud = base64Decode(secret.data!.cloud);

    const tenantID = cloud.match(/AZURE_TENANT_ID=([^\n]+)/);
    const clientID = cloud.match(/AZURE_CLIENT_ID=([^\n]+)/);
    const clientSecret = cloud.match(/AZURE_CLIENT_SECRET=([^\n]+)/);
    const resourceGroup = cloud.match(/AZURE_RESOURCE_GROUP=([^\n]+)/);
    const cloudName = cloud.match(/AZURE_CLOUD_NAME=([^\n]+)/);

    if (tenantID) {
      creds.tenantID = tenantID[1];
    }
    if (clientID) {
      creds.clientID = clientID[1];
    }
    if (clientSecret) {
      creds.clientSecret = clientSecret[1];
    }
    if (resourceGroup) {
      creds.resourceGroup = resourceGroup[1];
    }
    if (cloudName) {
      creds.cloudName = cloudName[1] as AzureCloudName;
    }

    return creds;
  } catch (e) {
    throw e;
  }
}

async function awsCredentialsSecret(corev1: CoreV1Api, namespace: string, aws: SnapshotStoreS3AWS): Promise<V1Secret|null> {
  let accessKeySecret = aws.accessKeySecret;
  if (accessKeySecret === redacted) {
    ({ accessKeySecret } = await readAWSCredentialsSecret(corev1, namespace));
  }

  if (!accessKeySecret && !aws.accessKeyID) {
    try {
      await corev1.deleteNamespacedSecret(awsSecretName, namespace);
    } catch(e) {
      if (e.response && e.response.statusCode === 404) {
        return null;
      }
      throw new ReplicatedError(`Failed to delete secret ${awsSecretName} from namespace ${namespace}. Velero will continue using the credentials in the secret if it exists rather than EC2 instance profiles`);
    }
    return null;
  }

  return {
    apiVersion: "v1",
    kind: "Secret",
    metadata: {
      name: awsSecretName,
      namespace,
    },
    stringData: {
      cloud: `[default]
aws_access_key_id=${aws.accessKeyID}
aws_secret_access_key=${accessKeySecret}`,
    },
  };
}

interface AWSCreds {
  accessKeyID?: string,
  accessKeySecret?: string,
}
// tslint:disable-next-line cyclomatic-complexity
async function readAWSCredentialsSecret(corev1: CoreV1Api, namespace: string): Promise<AWSCreds> {
  const creds: AWSCreds = {};

  let secret: any;
  try {
    const  { body } = await corev1.readNamespacedSecret(awsSecretName, namespace);
    secret = body;
  } catch (e) {
    if (e && e.response && e.response.statusCode === 404) {
      return {};
    }
    throw e;
  }
  if (!secret.data!.cloud) {
    return {};
  }
  const cloud = base64Decode(secret.data!.cloud);

  const keyID = cloud.match(/aws_access_key_id=([^\n]+)/);
  const keySecret = cloud.match(/aws_secret_access_key=([^\n]+)/);
  if (keyID) {
    creds.accessKeyID = keyID[1];
  }
  if (keySecret) {
    creds.accessKeySecret = keySecret[1];
  }

  return creds;
}

async function googleCredentialsSecret(corev1: CoreV1Api, namespace: string, google: SnapshotStoreGoogle): Promise<V1Secret> {
  let serviceAccount = google.serviceAccount;
  if (serviceAccount === redacted) {
    const sa = await readGoogleCredentialsSecret(corev1, namespace);
    if (sa) {
      serviceAccount = sa;
    } else {
      serviceAccount = "";
    }
  }

  return {
    apiVersion: "v1",
    kind: "Secret",
    metadata: {
      name: googleSecretName,
      namespace,
    },
    stringData: {
      cloud: serviceAccount,
    },
  }
}

async function readGoogleCredentialsSecret(corev1: CoreV1Api, namespace: string): Promise<string|void> {
  try {
    const { body: secret } = await corev1.readNamespacedSecret(googleSecretName, namespace);
    return base64Decode(secret.data!.cloud);
  } catch(e) {
    if (e.response && e.response.statusCode === 404) {
      return;
    }
    throw e;
  }
}
