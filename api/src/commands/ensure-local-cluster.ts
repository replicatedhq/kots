import * as util from "util";
import * as Api from "kubernetes-client";
import * as tmp from "tmp";
import * as fs from "fs";
import { exec } from "child_process";
import * as which from "which";

import { getPostgresPool } from "../util/persistence/db";
import { Params } from "../server/params";
import { ClusterStore } from "../cluster/cluster_store";
import { UserStore } from "../user/user_store";

const config = Api.config;

export const name = "ensure-local-cluster";
export const describe = "Ensure Local Cluster is provisioned";
export const builder = {

};

export const handler = async (argv) => {
  main(argv).catch((err) => {
    console.log(`Failed with error ${util.inspect(err)}`);
    process.exit(1);
  });
};

async function main(argv): Promise<any> {
  process.on('SIGTERM', function onSigterm () {
    process.exit();
  });

  if (!process.env["ENSURE_LOCAL_CLUSTER"]) {
    console.log("ENSURE_LOCAL_CLUSTER is not set, exiting");
    process.exit(0);
    return;
  }

  console.log(`Attempting to ensure local cluster is provisioned`);

  const pool = await getPostgresPool();
  const params = await Params.getParams();
  const clusterStore = new ClusterStore(pool, params);
  const userStore = new UserStore(pool);

  let localCluster = await clusterStore.getLocalShipOpsCluster();
  if (localCluster) {
    console.log(`Local cluster is already provisioned, all is well.`);
  } else {
    localCluster = await clusterStore.createNewCluster(undefined, true, "This Cluster", "ship");

    // Give all existing users access to this cluster
    const allUsers = await userStore.listAllUsers();
    for(const user of allUsers) {
      await clusterStore.addUserToCluster(localCluster.id!, user.id);
    }

    // Attempt to deploy this cluster
    const kubeconfig = config.getInCluster();

    const args: string[] = [];
    if (kubeconfig.url) {
      args.push(`--server=${kubeconfig.url}`);
    }
    if (kubeconfig.ca) {
      const caFile = tmp.fileSync();
      fs.writeFileSync(caFile.name, kubeconfig.ca);
      args.push(`--certificate-authority=${caFile.name}`);
    }
    if (kubeconfig.auth && kubeconfig.auth.bearer) {
      args.push(`--token=${kubeconfig.auth.bearer}`);
    }

    const clusterFile = tmp.fileSync();
    fs.writeFileSync(clusterFile.name, await clusterStore.getShipInstallationManifests(localCluster.id!));

    args.push("apply");
    args.push("-f");
    args.push(clusterFile.name)

    const kubectl = which.sync("kubectl");
    const fullCommand = [kubectl].concat(args).join(" ");

    // We run this twice becasue the yaml has both the crd and the resource itself
    // and kubectl doesn't interpret this properly all of the time
    console.log("applying crd");
    await runCommand(fullCommand);
    console.log("waiting");
    await sleep(5000);
    console.log("applying crd again");
    await runCommand(fullCommand);
  }

  process.exit(0);
}

async function runCommand(fullCommand) {
  const command = exec(fullCommand);
  command.stdout!.pipe(process.stdout);
  command.stderr!.pipe(process.stderr);

  return new Promise((resolve, reject) => {
    command.on("exit", (code) => {
      // We don't want to fail if code > 0 because sometimes rbac is preventing
      resolve();
    });
  });
}

async function sleep(ms) {
  return new Promise(resolve => {
    setTimeout(resolve, ms);
  });
}
