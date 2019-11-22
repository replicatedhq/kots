import { KotsAppStore } from "./kots_app_store";
import { ClusterStore } from "../cluster";
import tmp from "tmp";
import mkdirp from "mkdirp";
import path, { parse } from "path";
import fs, { mkdirSync } from "fs";
import { Params } from "../server/params";
import { kotsDecryptString } from "./kots_ffi";
import NodeGit from "nodegit";

interface commitTree {
  filename: string;
  contents: string;
}

export async function sendInitialGitCommitsForAppDownstream(kotsAppStore: KotsAppStore, clusterStore: ClusterStore, appId: string, clusterId: string): Promise<any> {
  const app = await kotsAppStore.getApp(appId);
  const cluster = await clusterStore.getCluster(clusterId);

  const gitOpsCreds = await kotsAppStore.getGitOpsCreds(appId, clusterId);

  const currentVersion = await kotsAppStore.getCurrentVersion(appId, clusterId);
  if (currentVersion) {
    const rendered = await app.render(`${currentVersion.parentSequence}`, `overlays/downstreams/${cluster.title}`);

    const tree: commitTree[] = [];
    tree.push({
      filename: "rendered.yaml",
      contents: rendered,
    });

    // TODO support -f and -k kubectl options
    await createGitCommit(gitOpsCreds, "master", tree);
  }

  const pendingVersions = await kotsAppStore.listPendingVersions(appId, clusterId);
  for (const pendingVersion of pendingVersions) {
    const rendered = await app.render(`${pendingVersion.parentSequence}`, `overlays/downstreams/${cluster.title}`);

    const tree: commitTree[] = [];
    tree.push({
      filename: "rendered.yaml",
      contents: rendered,
    });

    // TODO support -f and -k kubectl options
    await createGitCommit(gitOpsCreds, "master", tree);
  }
}

export async function createGitCommit(gitOpsCreds: any, branch: string, tree: commitTree[]): Promise<any> {
  const uriParts = gitOpsCreds.uri.split("/");
  const cloneUri = `git@github.com:${uriParts[3]}/${uriParts[4]}.git`;
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
    await NodeGit.Clone(cloneUri, localPath, cloneOptions);
    const repo = await NodeGit.Repository.open(localPath);

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

    const signature = NodeGit.Signature.now("KOTS Admin Console", "help@replicated.com");
    await repo.createCommit("HEAD", signature, signature, "commit goes here", oid, [parent]);

    const remote = await repo.getRemote("origin");
    const pushOptions = {
      callbacks: {
        credentials: async (url, username) => {
          const creds = await NodeGit.Cred.sshKeyMemoryNew(username, gitOpsCreds.pubKey, decryptedPrivateKey, "")
          return creds;
        }
      }
    };

    await remote.push([`refs/heads/${branch}:refs/heads/${branch}`], pushOptions);
  } catch (err) {
    console.log(err);
  } finally {
    // TODO delete
  }
}
