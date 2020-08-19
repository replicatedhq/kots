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
} from "@kubernetes/client-node";
import { logger } from "../../server/logger";
import { Etcd3 } from "etcd3";
import * as yaml from "js-yaml";

export function KurlMutations(stores: Stores, params: Params) {
  return {
    async deleteNode(root: any, { name }, context: Context): Promise<boolean> {
      context.requireSingleTenantSession();

      await purge(name);

      return false;
    },
  }
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
