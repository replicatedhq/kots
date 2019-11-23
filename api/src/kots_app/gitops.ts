import tmp from "tmp";
import path from "path";
import fs from "fs";
import { Params } from "../server/params";
import { kotsDecryptString } from "./kots_ffi";
import NodeGit from "nodegit";
import { Stores } from "../schema/stores";

interface commitTree {
  filename: string;
  contents: string;
}

export async function sendInitialGitCommitsForAppDownstream(stores: Stores, appId: string, clusterId: string): Promise<any> {
  const app = await stores.kotsAppStore.getApp(appId);
  const cluster = await stores.clusterStore.getCluster(clusterId);
  const downstreamGitOps = await stores.kotsAppStore.getDownstreamGitOps(appId, clusterId);

  const gitOpsCreds = await stores.kotsAppStore.getGitOpsCreds(appId, clusterId);

  const currentVersion = await stores.kotsAppStore.getCurrentVersion(appId, clusterId);
  if (currentVersion) {
    const rendered = await app.render(`${currentVersion.parentSequence}`, `overlays/downstreams/${cluster.title}`);

    const tree: commitTree[] = [];
    let filename = "";
    if (downstreamGitOps.path) {
      filename = path.join(downstreamGitOps.path, `${app.slug}.yaml`).substr(1);  // remove the leading /
    } else {
      filename = `${app.slug}.yaml`;
    }
    tree.push({
      filename,
      contents: rendered,
    });

    // TODO support -f and -k kubectl options
    if (downstreamGitOps.format === "single") {
      await createGitCommit(gitOpsCreds, downstreamGitOps.branch, tree, `Initial commit of ${app.name}`);
    } else {
      throw new Error("unsupported gitops format");
    }
  }

  const pendingVersions = await stores.kotsAppStore.listPendingVersions(appId, clusterId);
  for (const pendingVersion of pendingVersions) {
    const commitMessage = `Updating ${app.name} to version ${pendingVersion.sequence}`;
    await createGitCommitForVersion(stores, appId, clusterId, pendingVersion.parentSequence!, commitMessage);
  }
}

export async function createGitCommitForVersion(stores: Stores, appId: string, clusterId: string, parentSequence: number, commitMessage: string): Promise<any> {
  const app = await stores.kotsAppStore.getApp(appId);
  const cluster = await stores.clusterStore.getCluster(clusterId);
  const gitOpsCreds = await stores.kotsAppStore.getGitOpsCreds(app.id, cluster.id);
  const downstreamGitOps = await stores.kotsAppStore.getDownstreamGitOps(appId, clusterId);

  const rendered = await app.render(`${parentSequence}`, `overlays/downstreams/${cluster.title}`);

  const tree: commitTree[] = [];
  tree.push({
    filename: `${app.slug}.yaml`,
    contents: rendered,
  });

  // TODO support -f and -k kubectl options
  await createGitCommit(gitOpsCreds, downstreamGitOps.branch, tree, commitMessage);
}

export async function createGitCommit(gitOpsCreds: any, branch: string, tree: commitTree[], commitMessage: string): Promise<any> {
  const localPath = tmp.dirSync().name;
  const params = await Params.getParams();
  const decryptedPrivateKey = await kotsDecryptString(params.apiEncryptionKey, gitOpsCreds.privKey);
  const cloneOptions = {
    fetchOpts: {
      callbacks: {
        certificateCheck: () => { return 0; },
        credentials: async (url, username) => {
          const creds = await NodeGit.Cred.sshKeyMemoryNew(username, gitOpsCreds.pubKey, decryptedPrivateKey, "")
          return creds;
        }
      }
    },
  };
  try {
    await NodeGit.Clone(gitOpsCreds.cloneUri, localPath, cloneOptions);
    const repo = await NodeGit.Repository.open(localPath);

    // checkout/create branch
    if (branch !== "master") {
      let branchRef;
      try {
        branchRef = await NodeGit.Branch.lookup(repo, branch, NodeGit.Branch.BRANCH.REMOTE);
      } catch (err) {
        if (err.errno === -3) {
          // remote branch not found
          const head = await NodeGit.Reference.nameToId(repo, "HEAD");
          const parent = await repo.getCommit(head);
          branchRef =  await NodeGit.Branch.create(repo, branch, parent, false);
        }
      }
      await repo.checkoutBranch(branchRef, {});
    }

    // pull latest
    await repo.fetchAll({
      callbacks: {
        credentials: async (url, username) => {
          const creds = await NodeGit.Cred.sshKeyMemoryNew(username, gitOpsCreds.pubKey, decryptedPrivateKey, "")
          return creds;
        }
      }
    });
    await repo.mergeBranches(branch, `origin/${branch}`);
    
    // add files
    const index = await repo.refreshIndex();
    let output = localPath;
    for (const commitFile of tree) {
      const outputFile = path.join(output, commitFile.filename);
      const parsed = path.parse(outputFile);
      if (!fs.existsSync(parsed.dir)) {
        fs.mkdirSync(parsed.dir, {recursive: true});
      }
      fs.writeFileSync(path.join(output, commitFile.filename), commitFile.contents);
      await index.addByPath(commitFile.filename);
    }
    await index.write();

    const oid = await index.writeTree();
    const head = await NodeGit.Reference.nameToId(repo, "HEAD");
    const parent = await repo.getCommit(head);

    // commit
    const signature = NodeGit.Signature.now("KOTS Admin Console", "help@replicated.com");
    await repo.createCommit("HEAD", signature, signature, commitMessage, oid, [parent]);
    const remote = await repo.getRemote("origin");
    const pushOptions = {
      callbacks: {
        credentials: async (url, username) => {
          const creds = await NodeGit.Cred.sshKeyMemoryNew(username, gitOpsCreds.pubKey, decryptedPrivateKey, "")
          return creds;
        }
      }
    };
    
    // push
    await remote.push([`refs/heads/${branch}:refs/heads/${branch}`], pushOptions);
  } catch (err) {
    console.log(err);
  } finally {
    // TODO delete
  }
}
