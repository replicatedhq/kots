import tmp from "tmp";
import path from "path";
import fs from "fs";
import * as _ from "lodash";
import { Params } from "../server/params";
import { kotsDecryptString } from "./kots_ffi";
import NodeGit from "nodegit";
import { Stores } from "../schema/stores";
import { ReplicatedError } from "../server/errors";
import { getGitProviderCommitUrl } from "../util/utilities";

interface commitTree {
  filename: string;
  contents: string;
}

export async function sendInitialGitCommitsForAppDownstream(stores: Stores, appId: string, clusterId: string): Promise<any> {
  const app = await stores.kotsAppStore.getApp(appId);
  const downstreamGitOps = await stores.kotsAppStore.getDownstreamGitOps(appId, clusterId);
  
  const currentVersion = await stores.kotsAppStore.getCurrentVersion(appId, clusterId);
  if (currentVersion) {
    if (downstreamGitOps.format === "single") {
      await createGitCommitForVersion(stores, appId, clusterId, currentVersion.parentSequence!, `Initial commit of ${app.name}`);
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

export async function createGitCommitForVersion(stores: Stores, appId: string, clusterId: string, parentSequence: number, commitMessage: string): Promise<string> {
  const app = await stores.kotsAppStore.getApp(appId);
  const cluster = await stores.clusterStore.getCluster(clusterId);
  const gitOpsCreds = await stores.kotsAppStore.getGitOpsCreds(app.id, cluster.id);
  const downstreamGitOps = await stores.kotsAppStore.getDownstreamGitOps(appId, clusterId);

  const rendered = await app.render(`${parentSequence}`, `overlays/downstreams/${cluster.title}`);

  let filename = "";
  if (downstreamGitOps.path) {
    filename = path.join(downstreamGitOps.path, `${app.slug}.yaml`).replace(/^\//g, '');  // remove the leading /
  } else {
    filename = `${app.slug}.yaml`;
  }

  const tree: commitTree[] = [];
  tree.push({
    filename,
    contents: rendered,
  });

  // TODO support -f and -k kubectl options
  return await createGitCommit(gitOpsCreds, downstreamGitOps.branch, tree, commitMessage);
}

export async function createGitCommit(gitOpsCreds: any, branch: string, tree: commitTree[], commitMessage: string): Promise<string> {
  const localPath = tmp.dirSync().name;
  const params = await Params.getParams();
  const decryptedPrivateKey = await kotsDecryptString(params.apiEncryptionKey, gitOpsCreds.privKey);

  const options = {
    callbacks: {
      certificateCheck: () => { return 0; },
      credentials: async (url, username) => {
        const creds = await NodeGit.Cred.sshKeyMemoryNew(username, gitOpsCreds.pubKey, decryptedPrivateKey, "")
        return creds;
      }
    }
  }
  const cloneOptions = {
    fetchOpts: options
  };

  try {
    await NodeGit.Clone(gitOpsCreds.cloneUri, localPath, cloneOptions);
    const repo = await NodeGit.Repository.open(localPath);

    // create/checkout branch
    const references = await repo.getReferences();
    const branchRefName = `refs/heads/${branch}`;
    let isNewBranch = false;
    let branchRef = _.find(references, (reference: any) => reference.name() === branchRefName);
    if (!branchRef) {
      const branchRemoteRefName = `refs/remotes/origin/${branch}`;
      const branchRemoteRef = _.find(references, (reference: any) => reference.name() === branchRemoteRefName);
      if (branchRemoteRef) {
        // branch exists in remote (create locally)
        const parent = await repo.getBranchCommit(branchRemoteRef);
        branchRef = await repo.createBranch(branch, parent, false);
      } else {
        // branch does not exist
        const head = await NodeGit.Reference.nameToId(repo, "HEAD");
        const parent = await repo.getCommit(head);
        branchRef = await repo.createBranch(branch, parent, false);
        isNewBranch = true;
      }
    }
    await repo.checkoutRef(branchRef, {});

    // pull latest
    if (!isNewBranch) {
      await repo.fetchAll(options);
      await repo.mergeBranches(branch, `origin/${branch}`);
    }

    // add files
    const index = await repo.refreshIndex();
    let output = localPath;
    for (const commitFile of tree) {
      const outputFile = path.join(output, commitFile.filename);
      const parsed = path.parse(outputFile);
      if (!fs.existsSync(parsed.dir)) {
        fs.mkdirSync(parsed.dir, { recursive: true });
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
    const commitHash = await repo.createCommit("HEAD", signature, signature, commitMessage, oid, [parent]);

    // push
    const remote = await repo.getRemote("origin");
    await remote.push([`refs/heads/${branch}:refs/heads/${branch}`], options);

    return getGitProviderCommitUrl(gitOpsCreds.uri, commitHash, gitOpsCreds.provider);
  } catch (err) {
    throw new ReplicatedError(`Failed to create git commit ${err}`)
  }
}
