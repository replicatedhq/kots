import * as _ from "lodash";
import * as yaml from "js-yaml";
import { Stores } from "../schema/stores";
import { ReplicatedError } from "../server/errors";
import { getK8sNamespace, getKotsadmNamespace, kotsRenderFile } from "../kots_app/kots_ffi";
import { Backup } from "./velero";
import { backupStorageLocationName, VeleroClient } from "./resolvers/veleroClient";
import {
  kotsAppIdKey,
  kotsAppSlugKey,
  kotsAppSequenceKey,
  kotsClusterIdKey,
  kotsadmLabelKey,
  snapshotTriggerKey,
  SnapshotTrigger
} from "./snapshot";
import { logger } from "../server/logger";

export async function backup(stores: Stores, appId: string, scheduled: boolean) {
  const app = await stores.kotsAppStore.getApp(appId);
  const kotsVersion = await stores.kotsAppStore.getCurrentAppVersion(appId);
  if (!kotsVersion) {
    throw new ReplicatedError("App does not have a current version");
  }
  const clusters = await stores.clusterStore.listClustersForKotsApp(app.id);
  if (clusters.length !== 1) {
    throw new ReplicatedError("Must have exactly 1 cluster for backup");
  }
  const clusterId = clusters[0].id;

  let name = `manual-${Date.now()}`;
  if (scheduled) {
    name = `scheduled-${Date.now()}`;
  }

  const tmpl = await stores.snapshotsStore.getKotsBackupSpec(appId, kotsVersion.sequence);
  const rendered = await kotsRenderFile(app, stores, tmpl);
  const base = yaml.safeLoad(rendered) as Backup;
  const spec = (base && base.spec) || {};

  const namespaces = _.compact(spec.includedNamespaces);
  const deployNS = getK8sNamespace();
  if (namespaces.length === 0) {
    namespaces.push(deployNS);
  }

  const velero = new VeleroClient("velero"); // TODO namespace

  let backend: string;
  try {
    const backends = await velero.listBackends();
    if (_.includes(backends, backupStorageLocationName)) {
      backend = backupStorageLocationName;
    } else if (_.includes(backends, "local-ceph-rgw")) {
      backend = "local-ceph-rgw";
    } else {
      throw new ReplicatedError("No backupstoragelocation configured");
    }
  } catch (e) {
    throw e;
  }

  let backup: Backup = {
    apiVersion: "velero.io/v1",
    kind: "Backup",
    metadata: {
      name,
      annotations: {
        [snapshotTriggerKey]: scheduled ? SnapshotTrigger.Schedule : SnapshotTrigger.Manual,
        [kotsAppSlugKey]: app.slug,
        [kotsAppIdKey]: app.id,
        [kotsAppSequenceKey]: kotsVersion.sequence.toString(),
        [kotsClusterIdKey]: clusterId,
      }
    },
    spec: {
      hooks: spec.hooks,
      includedNamespaces: namespaces,
      ttl: convertTTL(app.snapshotTTL || ""),
      storageLocation: backend,
    }
  };

  const ownNS = getKotsadmNamespace();
  if (_.includes(namespaces, ownNS)) {
    // exclude kotsadm control plane objects
    backup.spec.labelSelector = {
      matchExpressions: [{
        key: kotsadmLabelKey,
        operator: "NotIn",
        values: ["kotsadm"],
      }],
    }
  }

  await velero.createBackup(backup);
}

export function convertTTL(ttl: string): string {
  const defaultTTL = "720h";
  const [quant, unit] = ttl.split(" ");
  const n = parseInt(quant);
  if (_.isNaN(n)) {
    logger.warn(`Ignoring invalid snapshot TTL: ${ttl}`);
    return defaultTTL;
  }
  switch (unit) {
  case "seoncds":
    return `${n}s`;
  case "minutes":
    return `${n}m`;
  case "hours":
    return `${n}h`;
  case "days":
    return `${n * 24}h`;
  case "weeks":
    return `${n * 168}h`;
  case "months":
    return `${n * 720}h`;
  case "years":
    return `${n * 8766}h`;
  default:
    logger.warn(`Ignoring invalid snapshot TTL: ${ttl}`);
  }

  return defaultTTL;
}
