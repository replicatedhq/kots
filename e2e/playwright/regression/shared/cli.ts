import * as semverjs from "semver";
import * as uuid from "uuid";

import {
  AWS_BUCKET_NAME,
  AWS_REGION,
  APP_SLUG
} from './constants';

import { execSync } from 'child_process';

export const deleteKurlConfigMap = (sshToAirgappedInstance?: string) => {
  runCommand(`kubectl delete configmap kurl-config --namespace kube-system --ignore-not-found`, sshToAirgappedInstance);
};

export type RegistryInfo = {
  ip: string;
  username: string;
  password: string;
};

export const getRegistryInfo = (isExistingCluster: boolean, sshToAirgappedInstance?: string): RegistryInfo => {
  let secretName = "registry-creds";

  if (isExistingCluster) {
    /**
     * this is a hack to work around the fact that kotsadm will automatically hide the registry settings in the airgap upload page if this secret exists
     * so we copy the secret with a different name and delete the old one
     */
    const secretYaml = runCommandWithOutput(`kubectl get secret ${secretName} -oyaml --ignore-not-found`, sshToAirgappedInstance);

    const newSecretName = "playwright-registry-creds";
    if(secretYaml !== "") {
      runCommand(`kubectl get secret ${secretName} -oyaml | sed s/'name: ${secretName}'/'name: ${newSecretName}'/ | kubectl apply -n default -f -`, sshToAirgappedInstance);
      runCommand(`kubectl delete secret ${secretName}`, sshToAirgappedInstance);
    }

    secretName = newSecretName;
  }

  const secretStr = runCommandWithOutput(`kubectl get secret ${secretName} -o=json`, sshToAirgappedInstance);
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

export const resetPassword = (namespace: string, isAirgapped: boolean, sshToAirgappedInstance: string) => {
  runCommand(`echo 'password' | kubectl kots reset-password -n ${namespace}`, sshToAirgappedInstance);
};

export const removeApp = (namespace: string, sshToAirgappedInstance: string) => {
  runCommand(`kubectl kots remove ${APP_SLUG} -n ${namespace} --force --undeploy`, sshToAirgappedInstance);
};

export const runCommand = (command: string, sshToAirgappedInstance?: string) => {
  if (sshToAirgappedInstance) {
    command = `${sshToAirgappedInstance} "${command}"`;
  }
  console.log(command, "\n");
  execSync(command, {stdio: 'inherit'});
};

export const runCommandWithOutput = (command: string, sshToAirgappedInstance?: string): string => {
  if (sshToAirgappedInstance) {
    command = `${sshToAirgappedInstance} "${command}"`;
  }
  console.log(command, "\n");
  return execSync(command).toString();
};
