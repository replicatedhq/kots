import * as semverjs from "semver";
import * as uuid from "uuid";

import {
  AWS_BUCKET_NAME,
  AWS_REGION,
  APP_SLUG,
  SNAPSHOTS_HOST_PATH,
  SSH_TO_JUMPBOX
} from './constants';

import { execSync } from 'child_process';

export const deleteKurlConfigMap = () => {
  runCommand(`kubectl delete configmap kurl-config --namespace kube-system --ignore-not-found`);
};

export type RegistryInfo = {
  ip: string;
  username: string;
  password: string;
};

export const getRegistryInfo = (isExistingCluster: boolean): RegistryInfo => {
  let secretName = "registry-creds";

  if (isExistingCluster) {
    /**
     * this is a hack to work around the fact that kotsadm will automatically hide the registry settings in the airgap upload page if this secret exists
     * so we copy the secret with a different name and delete the old one
     */
    const secretYaml = runCommandWithOutput(`kubectl get secret ${secretName} -oyaml --ignore-not-found`);

    const newSecretName = "playwright-registry-creds";
    if(secretYaml !== "") {
      runCommand(`kubectl get secret ${secretName} -oyaml | sed s/'name: ${secretName}'/'name: ${newSecretName}'/ | kubectl apply -n default -f -`);
      runCommand(`kubectl delete secret ${secretName}`);
    }

    secretName = newSecretName;
  }

  const secretStr = runCommandWithOutput(`kubectl get secret ${secretName} -o=json`);
  const parsedSecret = JSON.parse(secretStr);
  const dockerConfig = Buffer.from(parsedSecret.data[".dockerconfigjson"], "base64").toString("utf-8");
  const parsedDockerConfig = JSON.parse(dockerConfig);

  const auths = parsedDockerConfig.auths;
  const ip = Object.keys(auths)[0];
  
  return {
    ip,
    username: auths[ip].username, 
    password: auths[ip].password
  };
};

export const installVeleroAWS = (veleroVersion: string, veleroAwsPluginVersion: string) => {
  const isVelero10OrNewer = semverjs.gte(semverjs.coerce(veleroVersion), semverjs.coerce("1.10"));

  // delete velero namespace
  runCommand(`kubectl delete namespace velero --ignore-not-found`);

  // write creds to a file
  const credsFileName = "aws-creds.txt";
  runCommand(`cat >${credsFileName} <<EOL
[default]
aws_access_key_id = ${process.env.AWS_ACCESS_KEY_ID}
aws_secret_access_key = ${process.env.AWS_SECRET_ACCESS_KEY} 
EOL`);

  // download velero binary
  runCommand(`curl -LO https://github.com/vmware-tanzu/velero/releases/download/${veleroVersion}/velero-${veleroVersion}-linux-amd64.tar.gz && \
tar zxvf velero-${veleroVersion}-linux-amd64.tar.gz && \
sudo mv velero-${veleroVersion}-linux-amd64/velero /usr/local/bin/velero`);

  // install velero
  const prefix = uuid.v4();
  runCommand(`velero install \
    --provider aws \
    --plugins velero/velero-plugin-for-aws:${veleroAwsPluginVersion} \
    --bucket ${AWS_BUCKET_NAME} \
    --backup-location-config region=${AWS_REGION} \
    --snapshot-location-config region=${AWS_REGION} \
    --secret-file ${credsFileName} \
    --prefix ${prefix} \
    ${isVelero10OrNewer ? "--use-node-agent --uploader-type=restic" : "--use-restic"}`);
};

export const installVeleroHostPath = (
  veleroVersion: string,
  veleroAwsPluginVersion: string,
  registryInfo: RegistryInfo,
  isAirgapped: boolean
) => {
  // Delete velero namespace
  runCommand(`kubectl delete namespace velero --ignore-not-found`);

  if (isAirgapped) {
    prepareVeleroImages(veleroVersion, veleroAwsPluginVersion, registryInfo);
  }

  // Reset the host path directory for snapshots
  runCommand(`rm -rf ${SNAPSHOTS_HOST_PATH}`);
  runCommand(`mkdir -p ${SNAPSHOTS_HOST_PATH}`);
  runCommand(`chmod a+rwx ${SNAPSHOTS_HOST_PATH}`);

  const isVelero10OrNewer = semverjs.gte(semverjs.coerce(veleroVersion), semverjs.coerce("1.10"));

  // Download velero binary
  const veleroBinURL = `https://github.com/vmware-tanzu/velero/releases/download/${veleroVersion}/velero-${veleroVersion}-linux-amd64.tar.gz`;
  if (isAirgapped) {
    downloadViaJumpbox(veleroBinURL, `velero-${veleroVersion}-linux-amd64.tar.gz`);
  } else {
    runCommand(`curl -LO ${veleroBinURL}`);
  }

  // Extract
  runCommand(`tar zxvf velero-${veleroVersion}-linux-amd64.tar.gz && mv velero-${veleroVersion}-linux-amd64/velero velero`);

  // Install velero
  let installCommand = `./velero install \
    --no-default-backup-location \
    --no-secret \
    ${isVelero10OrNewer ? "--use-node-agent --uploader-type=restic" : "--use-restic"} \
    --use-volume-snapshots=false`;
  if (isAirgapped) {
    installCommand += ` \
    --image ${registryInfo.ip}/velero:${veleroVersion} \
    --plugins ${registryInfo.ip}/velero-plugin-for-aws:${veleroAwsPluginVersion}`;
  } else {
    installCommand += ` \
    --plugins velero/velero-plugin-for-aws:${veleroAwsPluginVersion}`;
  }
  runCommand(installCommand);

  // Configure hostpath backend
  let configureHostpathCommand = `yes | kubectl kots velero configure-hostpath --hostpath ${SNAPSHOTS_HOST_PATH} --namespace ${APP_SLUG}`;
  if (isAirgapped) {
    configureHostpathCommand += ` --kotsadm-registry ${registryInfo.ip} --kotsadm-namespace ${APP_SLUG} --registry-username ${registryInfo.username} --registry-password ${registryInfo.password}`;
  }
  runCommand(configureHostpathCommand);

  if (isAirgapped) {
    configureVeleroImagePullSecret(registryInfo);
  }

  // wait for velero to be ready
  waitForVeleroAndNodeAgent(60000);
}

export const prepareVeleroImages = (
  veleroVersion: string,
  veleroAwsPluginVersion: string,
  registryInfo: RegistryInfo
) => {
  const isVelero10OrNewer = semverjs.gte(semverjs.coerce(veleroVersion), semverjs.coerce("1.10"));

  /*
    we use skopeo (from the jumpbox) to copy the velero images from dockerhub to the registry on the airgapped instances.
  */

  console.log("Preparing velero images", "\n");

  // Create a NodePort service for the kurl registry so that we can copy images to it using skopeo from the jumpbox
  // Delete the service if it already exists
  runCommand(`kubectl --namespace kurl delete service registry-node --ignore-not-found`);
  runCommand(`cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: registry-node
  namespace: kurl
spec:
  type: NodePort
  ports:
  - nodePort: 30443
    port: 443
    protocol: TCP
    targetPort: 443 
  selector:
    app: registry
EOF`);

  // Copy velero image from docker to the registry
  runCommand(`skopeo copy docker://velero/velero:${veleroVersion} docker://${registryInfo.ip}:30443/velero:${veleroVersion} --dest-creds ${registryInfo.username}:${registryInfo.password} --dest-tls-verify=false`, true);

  // Copy velero aws plugin image from docker to the registry
  runCommand(`skopeo copy docker://velero/velero-plugin-for-aws:${veleroAwsPluginVersion} docker://${registryInfo.ip}:30443/velero-plugin-for-aws:${veleroAwsPluginVersion} --dest-creds ${registryInfo.username}:${registryInfo.password} --dest-tls-verify=false`, true);

  // Copy restore helper image from docker to the registry
  const restoreHelperImageName = isVelero10OrNewer ? "velero-restore-helper" : "velero-restic-restore-helper";
  runCommand(`skopeo copy docker://velero/${restoreHelperImageName}:${veleroVersion} docker://${registryInfo.ip}:30443/${restoreHelperImageName}:${veleroVersion} --dest-creds ${registryInfo.username}:${registryInfo.password} --dest-tls-verify=false`, true);

  // Create velero namespace so that applying the restore helper configmap doesn't fail.
  // This could be done after velero is installed, but it is easier to have it as part of the "prepare velero images" section.
  runCommand(`kubectl create namespace velero`);

  // Create restore helper configmap
  runCommand(`cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: ${isVelero10OrNewer ? "fs-restore-action-config" : "restic-restore-action-config"}
  namespace: velero
  labels:
    velero.io/plugin-config: ''
    ${isVelero10OrNewer ? "velero.io/pod-volume-restore: RestoreItemAction" : "velero.io/restic: RestoreItemAction"}
data:
  image: ${registryInfo.ip}/${restoreHelperImageName}:${veleroVersion}
EOF`);
};

const configureVeleroImagePullSecret = (registryInfo: RegistryInfo) => {
  // delete secret from velero namespace
  runCommand(`kubectl -n velero delete secret registry-creds --ignore-not-found`);

  // create secret in velero namespace from registry info
  runCommand(`kubectl -n velero create secret docker-registry registry-creds --docker-server=${registryInfo.ip} --docker-username=${registryInfo.username} --docker-password=${registryInfo.password}`);

  // patch velero deployment
  runCommand(`kubectl -n velero patch deployment velero --type=merge --patch='{"spec":{"template":{"spec":{ "imagePullSecrets":[{"name":"registry-creds"}] }}}}'`);
};

export const waitForVeleroAndNodeAgent = async (timeout: number = 300000): Promise<void> => {
  const startTime = Date.now();
  while (Date.now() - startTime < timeout) {
    if (isVeleroReady() && isNodeAgentReady()) {
      return;
    }
    await new Promise(resolve => setTimeout(resolve, 2000)); // Wait 2 seconds between checks
  }
  throw new Error(`Timeout waiting for Velero and Node Agent to be ready after ${timeout/1000} seconds`);
};

const isVeleroReady = (): boolean => {
  const veleroDeployment = runCommandWithOutput(`kubectl get deployment velero -n velero -ojson`);
  const parsedDeployment = JSON.parse(veleroDeployment);

  if (parsedDeployment.status.observedGeneration !== parsedDeployment.metadata.generation) {
    console.log(`observedGeneration: ${parsedDeployment.status.observedGeneration}, generation: ${parsedDeployment.metadata.generation}`);
    return false;
  }
  if (parsedDeployment.status.readyReplicas !== parsedDeployment.spec.replicas) {
    console.log(`readyReplicas: ${parsedDeployment.status.readyReplicas}, replicas: ${parsedDeployment.spec.replicas}`);
    return false;
  }
  if (!!parsedDeployment.status.unavailableReplicas) {
    console.log(`unavailableReplicas: ${parsedDeployment.status.unavailableReplicas}`);
    return false;
  }

  return true;
};

const isNodeAgentReady = (): boolean => {
  const daemonsetName = runCommandWithOutput(`kubectl get ds -n velero | awk 'NR>1 {print $1}' | tr -d '\n'`);
  const daemonset = runCommandWithOutput(`kubectl get ds ${daemonsetName} -n velero -ojson`);
  const parsedDaemonset = JSON.parse(daemonset);

  if (parsedDaemonset.status.observedGeneration !== parsedDaemonset.metadata.generation) {
    console.log(`observedGeneration: ${parsedDaemonset.status.observedGeneration}, generation: ${parsedDaemonset.metadata.generation}`);
    return false;
  }
  if (parsedDaemonset.status.currentNumberScheduled !== parsedDaemonset.status.desiredNumberScheduled) {
    console.log(`currentNumberScheduled: ${parsedDaemonset.status.currentNumberScheduled}, desiredNumberScheduled: ${parsedDaemonset.status.desiredNumberScheduled}`);
    return false;
  }
  if (parsedDaemonset.status.numberAvailable !== parsedDaemonset.status.desiredNumberScheduled) {
    console.log(`numberAvailable: ${parsedDaemonset.status.numberAvailable}, desiredNumberScheduled: ${parsedDaemonset.status.desiredNumberScheduled}`);
    return false;
  }
  if (parsedDaemonset.status.numberReady !== parsedDaemonset.status.desiredNumberScheduled) {
    console.log(`numberReady: ${parsedDaemonset.status.numberReady}, desiredNumberScheduled: ${parsedDaemonset.status.desiredNumberScheduled}`);
    return false;
  }

  return true;
};

export const cliAirgapInstall = (
  registryInfo: RegistryInfo,
  airgapBundlePath: string,
  licenseFile: string,
  configValuesFile: string,
  namespace: string,
  isMinimalRBAC: boolean
) => {
  try {
    runCommand(`kubectl kots install ${APP_SLUG} \
      --kotsadm-namespace ${APP_SLUG} \
      --kotsadm-registry ${registryInfo.ip} \
      --registry-username ${registryInfo.username} \
      --registry-password ${registryInfo.password} \
      --airgap-bundle ${airgapBundlePath} \
      --license-file ${licenseFile} \
      --config-values ${configValuesFile} \
      --namespace ${namespace} \
      --shared-password password \
      --port-forward=false`);
  } catch (error) {
    if (!isMinimalRBAC) {
      throw error;
    }
    console.log("Expected non-zero exit code in minimal RBAC due to preflight check errors. Continuing...");
  }

  runCommand(`cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: kotsadm-external
  namespace: ${namespace}
spec:
  type: NodePort
  selector:
    app: kotsadm
  ports:
  - port: 8800
    targetPort: 3000
    nodePort: 8800
EOF`);
};

export const cliAirgapUpdate = (
  newBundlePath: string,
  namespace: string,
  isExistingCluster: boolean,
  registryInfo?: RegistryInfo
) => {
  let upgradeCommand = `kubectl kots upstream upgrade ${APP_SLUG} --airgap-bundle ${newBundlePath} -n ${namespace}`;
  if (isExistingCluster) {
    upgradeCommand += ` --kotsadm-namespace ${APP_SLUG} --kotsadm-registry ${registryInfo?.ip} --registry-username ${registryInfo?.username} --registry-password ${registryInfo?.password}`;
  }
  runCommand(upgradeCommand);
};

export const resetPassword = (namespace: string) => {
  runCommand(`echo 'password' | kubectl kots reset-password -n ${namespace}`);
};

export const removeApp = (namespace: string) => {
  runCommand(`kubectl kots remove ${APP_SLUG} -n ${namespace} --force --undeploy`);
};

export const removeKots = (namespace: string) => {
  runCommand(`kubectl delete namespace ${namespace} --ignore-not-found`);
  runCommand(`kubectl delete clusterrole kotsadm-role kotsadm-operator-role --ignore-not-found`);
  runCommand(`kubectl delete clusterrolebinding kotsadm-rolebinding kotsadm-operator-rolebinding --ignore-not-found`);
};

export const downloadViaJumpbox = (remoteUrl: string, localPath: string) => {
  const command = `${SSH_TO_JUMPBOX} "curl -L '${remoteUrl}'" > ${localPath}`;
  console.log(command, "\n");
  execSync(command, {stdio: 'inherit'});
};

export const runCommand = (command: string, runOnJumpbox: boolean = false) => {
  if (runOnJumpbox) {
    command = `${SSH_TO_JUMPBOX} "${command}"`;
  }
  console.log(command, "\n");
  execSync(command, {stdio: 'inherit'});
};

export const runCommandWithOutput = (command: string, runOnJumpbox: boolean = false): string => {
  if (runOnJumpbox) {
    command = `${SSH_TO_JUMPBOX} "${command}"`;
  }
  console.log(command, "\n");
  return execSync(command).toString();
};
