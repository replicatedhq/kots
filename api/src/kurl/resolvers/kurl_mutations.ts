import * as _ from "lodash";
import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";
import { ReplicatedError } from "../../server/errors";
import { Context } from "../../context";
import {
  KubeConfig,
  CoreV1Api,
  V1beta1Eviction,
  V1OwnerReference,
  V1Pod } from "@kubernetes/client-node";
import { logger } from "../../server/logger";

export function KurlMutations(stores: Stores, params: Params) {
  return {
    async drainNode(root: any, { name }, context: Context) {
      context.requireSingleTenantSession();

      await drain(name);

      return false;
    },

    async deleteNode(root: any, { name }, context: Context) {
      context.requireSingleTenantSession();

      return false;
    }
  }
}

export interface drainResult {
  waitAndRetry: boolean;
  misconfiguredPolicyDisruptionBudget: boolean;
}

async function drain(name: string) {
  const kc = new KubeConfig();
  kc.loadFromDefault();
  const coreV1Client: CoreV1Api = kc.makeApiClient(CoreV1Api);

  // cordon the node
  let { response, body: node } = await coreV1Client.readNode(name);

  if (response.statusCode !== 200) {
    throw new ReplicatedError("Node not found");
  }

  if (!node.spec) {
    throw new ReplicatedError("Node spec not found");
  }

  node.spec.unschedulable = true;

  ({ response, body: node } = await coreV1Client.replaceNode(name, node));

  if (response.statusCode !== 200) {
    throw new ReplicatedError(`Cordon node: ${response.statusCode}`);
  }

  // list and evict pods
  let waitAndRetry = false;
  let misconfiguredPolicyDisruptionBudget = false;
  const labelSelectors = [
    // Defer draining self and pods that provide cluster services to other pods
    "app notin (rook-ceph-mon,rook-ceph-osd,rook-ceph-operator,kotsadm-api),k8s-app!=kube-dns",
    // Drain Rook pods
    "app in (rook-ceph-mon,rook-ceph-osd,rook-ceph-operator)",
    // Drain dns pod
    "k8s-app=kube-dns",
    // Drain self
    "app=kotsadm-api",
  ];
  for (let i = 0; i < labelSelectors.length; i++) {
    const labelSelector = labelSelectors[i];
    const fieldSelector = `spec.nodeName=${name}`;
    const { response, body: pods } = await coreV1Client.listPodForAllNamespaces(undefined, undefined, fieldSelector, labelSelector);

    if (response.statusCode !== 200) {
      throw new Error(`List pods response status: ${response.statusCode}`);
    }

    logger.debug(`Found ${pods.items.length} pods matching labelSelector ${labelSelector}`);
    for (let j = 0; j < pods.items.length; j++) {
      const pod = pods.items[j];

      if (shouldDrain(pod)) {
        const result = await evict(coreV1Client, pod);

        waitAndRetry = waitAndRetry || result.waitAndRetry;
        misconfiguredPolicyDisruptionBudget = misconfiguredPolicyDisruptionBudget || result.misconfiguredPolicyDisruptionBudget;
      }
    }
  }

  return { waitAndRetry, misconfiguredPolicyDisruptionBudget };
}

async function evict(coreV1Client: CoreV1Api, pod: V1Pod): Promise<drainResult> {
  const name = _.get(pod, "metadata.name", "");
  const namespace = _.get(pod, "metadata.namespace", "");
  const eviction: V1beta1Eviction = {
    apiVersion: "policy/v1beta1",
    kind: "Eviction",
    metadata: {
      name,
      namespace,
    },
  };

  logger.info(`Evicting pod ${name} in namespace ${namespace} from node ${_.get(pod, "spec.nodeName")}`);

  const result = { waitAndRetry: false, misconfiguredPolicyDisruptionBudget: false };

  const { response } = await coreV1Client.createNamespacedPodEviction(name, namespace, eviction);
  switch (response.statusCode) {
  case 200: // fallthrough
  case 201:
    return result;
  case 429:
    logger.warn(`Failed to delete pod ${name}: 429: PodDisruptionBudget is preventing deletion`);
    result.waitAndRetry = true;
    return result;
  case 500:
    // Misconfigured, e.g. included in multiple budgets
    logger.error(`Failed to evict pod ${name}: 500: possible PodDisruptionBudget misconfiguration`);
    result.misconfiguredPolicyDisruptionBudget = true;
    return result;
  default:
    throw new Error(`Unexpected response code ${response.statusCode}`);
  }
}

function shouldDrain(pod: V1Pod): boolean {
  // completed pods always ok to drain
  if (isFinished(pod)) {
    return true;
  }

  if (isMirror(pod)) {
    logger.info(`Skipping drain of mirror pod ${_.get(pod, "metadata.name")} in namespace ${_.get(pod, "metadata.namespace")}`);
    return false;
  }
  // TODO if orphaned it's ok to delete the pod
  if (isDaemonSetPod(pod)) {
    logger.info(`Skipping drain of DaemonSet pod ${_.get(pod, "metadata.name")} in namespace ${_.get(pod, "metadata.namespace")}`);
    return false;
  }

  return true;
}

function isFinished(pod: V1Pod): boolean {
  const phase = _.get(pod, "status.phase");
  // https://github.com/kubernetes/api/blob/5524a3672fbb1d8e9528811576c859dbedffeed7/core/v1/types.go#L2414
  const succeeded = "Succeeded";
  const failed = "Failed";

  return phase === succeeded || phase === failed;
}

function isMirror(pod: V1Pod): boolean {
  const mirrorAnnotation = "kubernetes.io/config.mirror";
  const annotations = pod.metadata && pod.metadata.annotations;

  return annotations ? _.has(annotations, mirrorAnnotation) : false;
}

function isDaemonSetPod(pod: V1Pod): boolean {
  return _.some(_.get(pod, "metadata.ownerReferences", []), (ownerRef: V1OwnerReference) => {
    return ownerRef.controller && ownerRef.kind === "DaemonSet";
  });
}
