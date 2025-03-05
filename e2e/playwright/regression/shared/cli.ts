import * as semverjs from "semver";
import * as uuid from "uuid";

import {
  AWS_ACCESS_KEY_ID,
  AWS_BUCKET_NAME,
  AWS_REGION
} from './constants';

const { execSync } = require("child_process");

export const deleteKurlConfigMap = (isAirgapped: boolean, sshToAirgappedInstance?: string) => {
  let deleteConfigmapCommand = `kubectl delete configmap kurl-config --namespace kube-system --ignore-not-found`;
  if (isAirgapped) {
    deleteConfigmapCommand = `${sshToAirgappedInstance} "${deleteConfigmapCommand}"`;
  }
  console.log(deleteConfigmapCommand, "\n");
  execSync(deleteConfigmapCommand, {stdio: 'inherit'});
};

export type RegistryInfo = {
  ip: string;
  username: string;
  password: string;
};

export const getRegistryInfo = (isAirgapped: boolean, isExistingCluster: boolean, sshToAirgappedInstance?: string): RegistryInfo => {
  let secretName = "registry-creds";

  if (isExistingCluster) {
    /**
     * this is a hack to work around the fact that kotsadm will automatically hide the registry settings in the airgap upload page if this secret exists
     * so we copy the secret with a different name and delete the old one
     */
    let getSecretCommand = `kubectl get secret ${secretName} -oyaml --ignore-not-found`;
    if (isAirgapped) {
      getSecretCommand = `${sshToAirgappedInstance} "${getSecretCommand}"`;
    }
    console.log(getSecretCommand, "\n");
    const secretYaml = execSync(getSecretCommand).toString();

    const newSecretName = "playwright-registry-creds";
    if(secretYaml !== "") {
      let copySecretCommand = `kubectl get secret ${secretName} -oyaml | sed s/'name: ${secretName}'/'name: ${newSecretName}'/ | kubectl apply -n default -f -`;
      if (isAirgapped) {
        copySecretCommand = `${sshToAirgappedInstance} "${copySecretCommand}"`;
      }
      console.log(copySecretCommand, "\n");
      execSync(copySecretCommand, {stdio: 'inherit'});

      let deleteSecretCommand = `kubectl delete secret ${secretName}`;
      if (isAirgapped) {
        deleteSecretCommand = `${sshToAirgappedInstance} "${deleteSecretCommand}"`;
      }
      console.log(deleteSecretCommand, "\n");
      execSync(deleteSecretCommand, {stdio: 'inherit'});
    }

    secretName = newSecretName;
  }

  let getCredsSecretCommand = `kubectl get secret ${secretName} -o=json`;
  if (isAirgapped) {
    getCredsSecretCommand = `${sshToAirgappedInstance} "${getCredsSecretCommand}"`;
  }
  console.log(getCredsSecretCommand, "\n");
  const secretStr = execSync(getCredsSecretCommand).toString();
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
  const deleteNSCommand = `kubectl delete namespace velero --ignore-not-found`;
  console.log(deleteNSCommand, "\n");
  execSync(deleteNSCommand, {stdio: 'inherit'});

  // write creds to a file
  const credsFileName = "aws-creds.txt";
  const credsCommand = `cat >${credsFileName} <<EOL
[default]
aws_access_key_id = ${AWS_ACCESS_KEY_ID}
aws_secret_access_key = ${process.env.AWS_SECRET_ACCESS_KEY} 
EOL`;
  execSync(credsCommand, {stdio: 'inherit'});

  // download velero binary
  const downloadCommand = `curl -LO https://github.com/vmware-tanzu/velero/releases/download/${veleroVersion}/velero-${veleroVersion}-linux-amd64.tar.gz && \
tar zxvf velero-${veleroVersion}-linux-amd64.tar.gz && \
sudo mv velero-${veleroVersion}-linux-amd64/velero /usr/local/bin/velero`;
  console.log(downloadCommand, "\n");
  execSync(downloadCommand, {stdio: 'inherit'});

  // install velero
  const prefix = uuid.v4();
  const installCommand = `velero install \
    --provider aws \
    --plugins velero/velero-plugin-for-aws:${veleroAwsPluginVersion} \
    --bucket ${AWS_BUCKET_NAME} \
    --backup-location-config region=${AWS_REGION} \
    --snapshot-location-config region=${AWS_REGION} \
    --secret-file ${credsFileName} \
    --prefix ${prefix} \
    ${isVelero10OrNewer ? "--use-node-agent --uploader-type=restic" : "--use-restic"}
`;
  console.log(installCommand, "\n");
  execSync(installCommand, {stdio: 'inherit'});
};

export const resetPassword = (namespace: string, isAirgapped: boolean, sshToAirgappedInstance: string) => {
  let resetCommand = `echo 'password' | kubectl kots reset-password -n ${namespace}`;
  if (isAirgapped) {
    resetCommand = `${sshToAirgappedInstance} "${resetCommand}"`;
  }
  console.log(resetCommand, "\n");
  execSync(resetCommand, {stdio: 'inherit'});
};

export const runCommand = (command: string, isAirgapped: boolean, sshToAirgappedInstance?: string) => {
  if (isAirgapped) {
    command = `${sshToAirgappedInstance} "${command}"`;
  }
  console.log(command, "\n");
  execSync(command, {stdio: 'inherit'});
};
