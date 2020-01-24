import * as _ from "lodash";
import * as yaml from "js-yaml";
import {
  AppsV1Api,
  BatchV1beta1Api,
  KubeConfig,
  V1beta1CronJob,
  V1beta1CronJobSpec,
  V1Deployment } from "@kubernetes/client-node";
import { getKotsadmNamespace } from "../kots_app/kots_ffi";
import { ReplicatedError } from "../server/errors";
import { logger } from "../server/logger";
import { kotsadmLabelKey } from "./snapshot";

function snapshotScheduleName(appSlug: string) {
  return `velero-${appSlug}`;
}

const scheduleSelectionKey = "kots.io/snapshot-schedule-selection";
const kotsadmLabelValue = "kotsadm";

export async function deleteSchedule(appSlug: string): Promise<void> {
  const name = snapshotScheduleName(appSlug);
  const ownNS = getKotsadmNamespace();
  const kc = new KubeConfig();
  kc.loadFromDefault();
  const batchv1 = kc.makeApiClient(BatchV1beta1Api);

  try {
    await batchv1.deleteNamespacedCronJob(name, ownNS);
  } catch (e) {
    if (e.response && e.response.statusCode === 404) {
      return;
    }
    throw e;
  }
}

export async function schedule(appSlug: string, schedule: string, selection: string): Promise<void> {
  const kc = new KubeConfig();
  kc.loadFromDefault();
  const batchv1 = kc.makeApiClient(BatchV1beta1Api);
  const appsv1 = kc.makeApiClient(AppsV1Api);

  const cronJobName = snapshotScheduleName(appSlug);
  const labels = { kotsadmLabelKey: kotsadmLabelValue };
  const ownNS = getKotsadmNamespace();
  const ownImage = await getOwnImage(kc.makeApiClient(AppsV1Api), ownNS);
  const spec: V1beta1CronJobSpec = {
    concurrencyPolicy: "Forbid",
    schedule: schedule,
    startingDeadlineSeconds: 30,
    jobTemplate: {
      metadata: { labels },
      spec: {
        template: {
          metadata: { labels },
          spec: {
            containers: [{
              name: cronJobName,
              image: ownImage,
              command: [
                "/bin/bash", 
                "-c",
                `curl -v --fail -X POST http://kotsadm-api:3000/api/v1/kots/${appSlug}/snapshot`,
              ],
            }],
            restartPolicy: "OnFailure",
          },
        },
      },
    },
  };

  try {
    let { body: cronJob } = await batchv1.readNamespacedCronJob(cronJobName, ownNS);
    cronJob.spec = spec;
    cronJob.metadata!.annotations = cronJob.metadata!.annotations || {};
    cronJob.metadata!.annotations[scheduleSelectionKey] = selection;

    await batchv1.replaceNamespacedCronJob(cronJobName, ownNS, cronJob);
    return;
  } catch (e) {
    const statusCode = e.response && e.response.statusCode;
    if (statusCode === 403) {
      throw  new ReplicatedError(`Forbidden: RBAC may be misconfigured for reading or updating CronJobs in namespace ${ownNS}`);
    }
    if (statusCode !== 404) {
      if (e.response && e.response.body && e.response.body.message) {
        throw new ReplicatedError(e.response.body.message);
      }
      throw e;
    }
  }

  const cj: V1beta1CronJob = {
    apiVersion: "batch/v1beta1",
    kind: "CronJob",
    metadata: {
      name: cronJobName,
      namespace: ownNS,
      labels,
      annotations: {
        [scheduleSelectionKey]: selection,
      },
    },
    spec,
  };

  try {
    await batchv1.createNamespacedCronJob(ownNS, cj);
  } catch (e) {
    if (e.response && e.response.statusCode === 403) {
      throw new ReplicatedError(`Forbidden: RBAC may be misconfigured for creating CronJobs in namespace ${ownNS}`);
    }
    if (e.response && e.response.body && e.response.body.message) {
      throw new ReplicatedError(e.response.body.message);
    }
    throw e;
  }
}

async function getOwnImage(appsv1: AppsV1Api, ownNS: string): Promise<string> {
  let deployment: V1Deployment;

  try {
    ({ body: deployment } = await appsv1.readNamespacedDeployment("kotsadm-api", ownNS));
  } catch(e) {
    if (e.statusCode === 403) {
      throw new ReplicatedError(`Failed to lookup image from kotsadm-api deployment in namespace ${ownNS}: 403`);
    }
    throw e;
  }

  for (let container of deployment.spec!.template!.spec!.containers) {
    if (container.name === "kotsadm-api") {
      return container.image!;
    }
  }

  throw  new ReplicatedError(`Failed to lookup image from kotsadm-api deployment in namespace ${ownNS}`);
}

export interface scheduleSettings {
  schedule: string;
  selection: string;
}

export async function readSchedule(appSlug): Promise<scheduleSettings|null> {
  const kc = new KubeConfig();
  kc.loadFromDefault();
  const batchv1 = kc.makeApiClient(BatchV1beta1Api);
  const cronJobName = snapshotScheduleName(appSlug);
  const ownNS = getKotsadmNamespace();

  try {
    const { response, body: cronJob } = await batchv1.readNamespacedCronJob(cronJobName, ownNS);

    return {
      schedule: cronJob!.spec!.schedule,
      selection: (cronJob.metadata!.annotations && cronJob.metadata!.annotations[scheduleSelectionKey]) || "custom",
    };
  } catch (e) {
    if (e.response && e.response.statusCode === 404) {
      return null;
    }
    throw e;
  }
  return null;
}
