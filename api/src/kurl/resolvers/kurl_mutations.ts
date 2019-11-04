import { inspect } from "util";
import http from "http";
import fs from "fs";
import * as _ from "lodash";
import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";
import { ReplicatedError } from "../../server/errors";
import { Context } from "../../context";
import {
  Exec,
  KubeConfig,
  AppsV1Api,
  CoreV1Api,
  V1beta1Eviction,
  V1Node,
  V1OwnerReference,
  V1Pod,
  V1Status,
  BatchV1Api,
  V1Job,
  V1PodSpec,
  V1Container,
  V1ConfigMap,
  VersionApi,
  V1JobSpec,
} from "@kubernetes/client-node";
import { logger } from "../../server/logger";
import { IncomingMessage } from "http";
import { Etcd3 } from "etcd3";
import * as yaml from "js-yaml";

export function KurlMutations(stores: Stores, params: Params) {
  return {
    async drainNode(root: any, { name }, context: Context): Promise<boolean> {
      context.requireSingleTenantSession();

      await drain(name);

      return false;
    },

    async deleteNode(root: any, { name }, context: Context): Promise<boolean> {
      context.requireSingleTenantSession();

      await purge(name);

      return false;
    },

    async generateWorkerAddNodeCommand(root: any, args: any, context: Context): Promise<Command> {
      return await generateWorkerAddNodeCommand();
    },
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

async function purge(name: string) {
  const kc = new KubeConfig();
  kc.loadFromDefault();
  const coreV1Client: CoreV1Api = kc.makeApiClient(CoreV1Api);
  const appsV1Client: AppsV1Api = kc.makeApiClient(AppsV1Api);
  const exec: Exec = new Exec(kc);

  let { response, body: node } = await coreV1Client.readNode(name);

  if (response.statusCode !== 200) {
    throw new ReplicatedError(`get node ${name} response: ${response.statusCode}`);
  }

  ({ response } = await coreV1Client.deleteNode(name));

  if (response.statusCode !== 200) {
    throw new Error(`Delete node returned status code ${response.statusCode}`);
  }

  await purgeOSD(coreV1Client, appsV1Client, exec, name);

  const isMaster = _.has(node.metadata!.labels!, "node-role.kubernetes.io/master");
  if (isMaster) {
    logger.debug(`Node ${name} is a master: running etcd and kubeadm endpoints purge steps`);
    await purgeMaster(coreV1Client, node);
  } else {
    logger.debug(`Node ${name} is not a master: skipping etcd and kubeadm endpoints purge steps`);
  }
}

async function purgeMaster(coreV1Client: CoreV1Api, node: V1Node) {
  const remainingMasterIPs: Array<string> = [];
  let purgedMasterIP = "";

  // 1. Remove the purged endpoint from kubeadm's list of API endpoints in the kubeadm-config
  // ConfigMap in the kube-system namespace. Keep the list of all master IPs for step 2.
  const configMapName = "kubeadm-config";
  const configMapNS = "kube-system";
  const configMapKey = "ClusterStatus";
  const { response: cmResponse, body: kubeadmCM } = await coreV1Client.readNamespacedConfigMap(configMapName, configMapNS);
  if (cmResponse.statusCode !== 200) {
    throw new Error(`Get kubeadm-config map from kube-system namespace: ${cmResponse.statusCode}`);
  }

  const clusterStatus = yaml.safeLoad(kubeadmCM.data![configMapKey]);
  _.each(clusterStatus.apiEndpoints, (obj, nodeName) => {
    if (nodeName === node.metadata!.name) {
      purgedMasterIP = obj.advertiseAddress;
    } else {
      remainingMasterIPs.push(obj.advertiseAddress);
    }
  });
  delete(clusterStatus.apiEndpoints[node.metadata!.name!]);
  kubeadmCM.data![configMapKey] = yaml.safeDump(clusterStatus);
  const { response: cmReplaceResp } = await coreV1Client.replaceNamespacedConfigMap(configMapName, configMapNS, kubeadmCM);

  if (!purgedMasterIP) {
    logger.warn(`Failed to find IP of deleted master node from kubeadm-config: skipping etcd peer removal step`);
    return;
  }
  if (!remainingMasterIPs.length) {
    logger.error(`Cannot remove etcd peer: no remaining etcd endpoints available to connect to`);
    return;
  }

  // 2. Use the credentials from the mounted etcd client cert secret to connect to the remaining
  // etcd members and tell them to forget the purged member.
  const etcd = new Etcd3({
    credentials: {
      rootCertificate: fs.readFileSync("/etc/kubernetes/pki/etcd/ca.crt"),
      privateKey: fs.readFileSync("/etc/kubernetes/pki/etcd/client.key"),
      certChain: fs.readFileSync("/etc/kubernetes/pki/etcd/client.crt"),
    },
    hosts: _.map(remainingMasterIPs, (ip) => `https://${ip}:2379`),
  });
  const peerURL = `https://${purgedMasterIP}:2380`;
  const { members } = await etcd.cluster.memberList();
  const purgedMember = _.find(members, (member) => {
    return _.includes(member.peerURLs, peerURL);
  });
  if (!purgedMember) {
    logger.info(`Purged node was not a member of etcd cluster`);
    return;
  }

  logger.info(`Removing etcd member ${purgedMember.ID} ${purgedMember.name}`);
  await etcd.cluster.memberRemove({ ID: purgedMember.ID });
}

async function purgeOSD(coreV1Client: CoreV1Api, appsV1Client: AppsV1Api, exec: Exec, nodeName: string): Promise<void> {
  const namespace = "rook-ceph";
  // 1. Find the Deployment for the OSD on the purged node and lookup its osd ID from labels
  // before deleting it.
  const osdLabelSelector = "app=rook-ceph-osd";
  let { response, body: deployments } = await appsV1Client.listNamespacedDeployment(namespace, undefined, undefined, undefined, undefined, osdLabelSelector);
  if (response.statusCode === 404) {
    return;
  }
  if (response.statusCode !== 200) {
    throw new Error(`List osd deployments in rook-ceph namespace returned status code ${response.statusCode}`);
  }

  let osdID = "";

  for (let i = 0; i < deployments.items.length; i++) {
    const deploy = deployments.items[i];
    const hostname = deploy.spec!.template!.spec!.nodeSelector!["kubernetes.io/hostname"]

    if (hostname === nodeName) {
      osdID = deploy.metadata!.labels!["ceph-osd-id"];
      logger.info(`Deleting OSD deployment on host ${nodeName}`);
      ({ response } = await appsV1Client.deleteNamespacedDeployment(deploy.metadata!.name!, namespace, undefined, undefined, undefined, undefined, "Background"));
      if (response.statusCode !== 200) {
        throw new Error(`Got response status code ${response.statusCode}`);
      }
      break;
    }
  }

  if (osdID === "") {
    logger.info("Failed to find ceph osd id for node");
    return;
  }

  logger.info(`Purging ceph OSD with id ${osdID}`);

  // 2. Using the osd ID discovered in step 1, exec into the Rook operator pod and run the ceph
  // command to purge the OSD
	const operatorLabelSelector = "app=rook-ceph-operator";
  let { response: resp, body: pods } = await coreV1Client.listNamespacedPod(namespace, undefined, undefined, undefined, undefined, operatorLabelSelector);

  if (resp.statusCode !== 200) {
    throw new Error(`List operator pod in rook-ceph namespace returned status code ${resp.statusCode}`);
  }

  if (pods.items.length !== 1) {
    logger.warn(`Found ${pods.items.length} rook operator pods: skipping osd purge command`);
    return;
  }
  const pod = pods.items[0];

  await new Promise((resolve, reject) => {
    exec.exec(
      "rook-ceph",
      pod.metadata!.name!,
      "rook-ceph-operator",
      ["ceph", "osd", "purge", osdID, "--force"],
      process.stdout,
      process.stderr,
      null,
      false,
      (v1Status: V1Status) => {
        if (v1Status.status === "Failure") {
          reject(new Error(v1Status.message));
          return;
        }
        resolve();
      },
    );
  })
}

function readAll(stream: IncomingMessage): Promise<string> {
  const chunks: Array<any> = [];

  return new Promise((resolve, reject) => {
    stream.on('data', chunk => chunks.push(chunk));
    stream.on('error', reject);
    stream.on('end', () => resolve(Buffer.concat(chunks).toString('utf8')));
  });
}

export interface Command {
  command: string[];
  expiry: number;
}

// master: airgap kubernetes-master-address=${KUBERNETES_API_ADDRESS} kubeadm-token=${BOOTSTRAP_TOKEN} kubeadm-token-ca-hash=$KUBEADM_TOKEN_CA_HASH kubernetes-version=$KUBERNETES_VERSION cert-key=${CERT_KEY} control-plane ${dockerRegistryIP}

// worker: kubernetes-master-address=${KUBERNETES_API_ADDRESS} kubeadm-token=${BOOTSTRAP_TOKEN} kubeadm-token-ca-hash=$KUBEADM_TOKEN_CA_HASH kubernetes-version=$KUBERNETES_VERSION ${dockerRegistryIP}

async function generateWorkerAddNodeCommand(): Promise<Command> {
  const kc = new KubeConfig();
  kc.loadFromDefault();

  const versionClient: VersionApi = kc.makeApiClient(VersionApi);
  const versionInfo = await versionClient.getCode();

  const kubernetesVersion = versionInfo.body.gitVersion;

  let data = await readKurlConfigMap();

  // if the token expires withing the period, regenerate it
  const regeneratePreiod = 10 * 60 * 1000; // 10 minutes
  const nowUnix = (new Date()).getTime();
  let bootstrapTokenExpiration = Date.parse(data.bootstrap_token_expiration);
  if (isNaN(bootstrapTokenExpiration)) {
    console.log(`Failed to parse bootstrap_token_expiration ${data.bootstrap_token_expiration}`);
    bootstrapTokenExpiration = 0;
  }
  if (nowUnix + regeneratePreiod > bootstrapTokenExpiration) {
    console.log(`Bootstrap token expired ${new Date(bootstrapTokenExpiration)}, regenerating`);
    try {
      await runKurlUtilJobAndWait(["/usr/local/bin/join"]);
      data = await readKurlConfigMap();
    } catch (err) {
      console.log(err);
      throw err;
    }
    bootstrapTokenExpiration = Date.parse(data.bootstrap_token_expiration);
  }

  // data.airgap
  // data.bootstrap_token
  // data.ca_hash
  // data.cert_key
  // data.docker_registry_ip
  // data.ha
  // data.installer_id
  // data.kubernetes_api_address
  // data.kurl_url
  // data.upload_certs_expiration

  const command = [
    `curl -sSL ${data.kurl_url}/${data.installer_id}/join.sh | sudo bash -s`,
    `kubernetes-master-address=${data.kubernetes_api_address}`,
    `kubeadm-token=${data.bootstrap_token}`,
    `kubeadm-token-ca-hash=${data.ca_hash}`,
    `docker-registry-ip=${data.docker_registry_ip}`,
    `kubernetes-version=${kubernetesVersion}`,
  ];

  console.log("Generated node join command", command);

  return {
    command: command,
    expiry: bootstrapTokenExpiration / 1000,
  };
}

async function readKurlConfigMap(): Promise<{ [ key: string]: string }> {
  const kc = new KubeConfig();
  kc.loadFromDefault();

  const coreV1Client: CoreV1Api = kc.makeApiClient(CoreV1Api);

  let response: http.IncomingMessage;
  let configMap: V1ConfigMap;

  try {
    ({ response, body: configMap } = await coreV1Client.readNamespacedConfigMap("kurl-config", "kube-system"));
  } catch (err) {
    throw new ReplicatedError(`Failed to read config map ${err.response && err.response.body ? err.response.body.message : ""}`);
  }

  if (response.statusCode !== 200 || !configMap) {
    throw new ReplicatedError(`Config map not found`);
  }

  if (!configMap.data) {
    throw new ReplicatedError("Config map data not found");
  }

  return configMap.data;
}

async function runKurlUtilJobAndWait(command: string[]) {
  const kc = new KubeConfig();
  kc.loadFromDefault();

  const batchV1Client: BatchV1Api = kc.makeApiClient(BatchV1Api);

  let job: V1Job;
  try {
    ({ body: job } = await batchV1Client.createNamespacedJob("kube-system", {
      apiVersion: "batch/v1",
      kind: "Job",
      metadata: {
        generateName: "kurl-util-join-",
      },
      spec: {
        completions: 1,
        backoffLimit: 3,
        ttlSecondsAfterFinished: 60,
        template: {
          metadata: {
            labels: {
              "app": "kurl-util-join",
            },
          },
          spec: {
            // nodeSelector: {"node-role.kubernetes.io/master": ""}, // TODO: this is needed for master join
            restartPolicy: "Never",
            activeDeadlineSeconds: 120,
            containers: [{
              name: "kurl-util-join",
              image: "replicated/kurl-util:latest",
              imagePullPolicy: "IfNotPresent",
              command: command,
              volumeMounts: [{
                name: "etc-kubernetes",
                mountPath: "/etc/kubernetes",
              }],
            }] as V1Container[],
            volumes: [{
              name: "etc-kubernetes",
              hostPath: {
                path: "/etc/kubernetes",
              },
            }],
          } as V1PodSpec,
        },
      } as V1JobSpec,
    }));
  } catch (err) {
    throw new ReplicatedError(`Failed to create job ${err.response && err.response.body ? err.response.body.message : ""}`);
  }

  if (!job.metadata || !job.metadata.name || !job.metadata.namespace) {
    throw new ReplicatedError("Job creation failed");
  }

  while (true) {
    try {
      let { body: jobStatus } = await batchV1Client.readNamespacedJobStatus(job.metadata.name, job.metadata.namespace);
      if (jobStatus.status) {
        if (jobStatus.status.succeeded) {
          console.log(`Job ${job.metadata.name} creation succeeded`);
          return;
        } else if (jobStatus.status.failed) {
          throw new ReplicatedError("job failed");
        }
      }
    } catch( err ) {
      console.log(`Failed to read job status ${err.response && err.response.body ? err.response.body.message : ""}`);
    }
    await sleep(2); // sleep 2 seconds
  }
}

async function sleep(seconds): Promise<void> {
  await new Promise(resolve => setTimeout(resolve, seconds * 1000));
}
