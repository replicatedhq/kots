import _ from "lodash";
import { generateKeyPairSync } from "crypto";
import sshpk from "sshpk";
import { Context } from "../../context";
import path from "path";
import tmp from "tmp";
import yaml from "js-yaml";
import NodeGit from "nodegit";
import { Stores } from "../../schema/stores";
import { Cluster } from "../../cluster";
import { ReplicatedError } from "../../server/errors";
import { kotsAppFromLicenseData, kotsFinalizeApp, kotsAppCheckForUpdates, kotsRewriteImagesInVersion, kotsAppDownloadUpdates } from "../kots_ffi";
import { Update } from "../kots_ffi";
import { KotsAppRegistryDetails } from "../kots_app"
import * as k8s from "@kubernetes/client-node";
import { kotsEncryptString, kotsDecryptString } from "../kots_ffi"
import { Params } from "../../server/params";
import { Repeater } from "../../util/repeater";
import { sendInitialGitCommitsForAppDownstream } from "../gitops";

export function KotsMutations(stores: Stores) {
  return {
    async setAppGitOps(root: any, args: any, context: Context): Promise<string> {
      const { appId, clusterId, gitOpsInput } = args;

      const app = await context.getApp(appId);

      const { publicKey, privateKey } = generateKeyPairSync("rsa", {
        modulusLength: 4096,
        publicKeyEncoding: {
          type: "pkcs1",
          format: "pem",
        },
        privateKeyEncoding: {
          type: "pkcs1",
          format: "pem",
        },
      });

      const params = await Params.getParams();
      const parsedPublic = sshpk.parseKey(publicKey, "pem");
      const sshPublishKey = parsedPublic.toString("ssh");

      const encryptedPrivateKey = await kotsEncryptString(params.apiEncryptionKey, privateKey);
      const gitopsRepo = await stores.kotsAppStore.createGitOpsRepo(gitOpsInput.provider, gitOpsInput.uri, encryptedPrivateKey, sshPublishKey);

      await stores.kotsAppStore.setAppDownstreamGitOpsConfiguration(app.id, clusterId, gitopsRepo.id, gitOpsInput.branch, gitOpsInput.path, gitOpsInput.format);

      return publicKey;
    },

    async updateAppGitOps(root: any, args: any, context: Context): Promise<string> {
      const { appId, clusterId, gitOpsInput } = args;

      const integrationToUpdate = await stores.kotsAppStore.getGitOpsCreds(appId, clusterId);

      let sshPublishKey = integrationToUpdate.keyPub;
      let encryptedPrivateKey = integrationToUpdate.keyPriv;
      if (integrationToUpdate.provider !== gitOpsInput.provider) {
        const { publicKey, privateKey } = generateKeyPairSync("rsa", {
          modulusLength: 4096,
          publicKeyEncoding: {
            type: "pkcs1",
            format: "pem",
          },
          privateKeyEncoding: {
            type: "pkcs1",
            format: "pem",
          },
        });
        const params = await Params.getParams();
        const parsedPublic = sshpk.parseKey(publicKey, "pem");
        sshPublishKey = parsedPublic.toString("ssh");
        encryptedPrivateKey = await kotsEncryptString(params.apiEncryptionKey, privateKey);
      }
      
      await stores.kotsAppStore.updateGitOpsRepo(integrationToUpdate.id, gitOpsInput.provider, gitOpsInput.uri, encryptedPrivateKey, sshPublishKey);

      await stores.kotsAppStore.setAppDownstreamGitOpsConfiguration(appId, clusterId, integrationToUpdate.id, gitOpsInput.branch, gitOpsInput.path, gitOpsInput.format);

      return sshPublishKey;
    },

    async checkForKotsUpdates(root: any, args: any, context: Context): Promise<number> {
      const { appId } = args;

      const app = await context.getApp(appId);
      const cursor = await stores.kotsAppStore.getMidstreamUpdateCursor(app.id);

      const updateStatus = await stores.kotsAppStore.getUpdateDownloadStatus();
      if (updateStatus.status === "running") {
        return 0;
      }

      const liveness = new Repeater(() => {
        return new Promise((resolve) => {
          stores.kotsAppStore.updateUpdateDownloadStatusLiveness().finally(() => {
            resolve();
          })
        });
      }, 1000);

      var updatesAvailable: Update[];
      try {
        await stores.kotsAppStore.setUpdateDownloadStatus("Checking for updates...", "running");

        liveness.start();

        updatesAvailable = await kotsAppCheckForUpdates(app, cursor);
      } catch(err) {
        liveness.stop();
        await stores.kotsAppStore.setUpdateDownloadStatus(String(err), "failed");
        throw err;
      }

      let downloadUpdates = async function() {
        try {
          await kotsAppDownloadUpdates(updatesAvailable, app, stores);

          await stores.kotsAppStore.clearUpdateDownloadStatus();
        } catch(err) {
          await stores.kotsAppStore.setUpdateDownloadStatus(String(err), "failed");
          throw err;
        } finally {
          liveness.stop();
        }
      }
      downloadUpdates(); // download asyncronously
      return updatesAvailable.length;
    },

    async testGitOpsConnection(root: any, args: any, context: Context) {
      const { appId, clusterId } = args;

      const gitOpsCreds = await stores.kotsAppStore.getGitOpsCreds(appId, clusterId);
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
        NodeGit.Repository.openBare(localPath);
        // TODO check if we have write access!

        await stores.kotsAppStore.setGitOpsError(gitOpsCreds.id, "");
        // Send current and pending versions to git
        // We need a persistent, durable queue for this to handle the api container
        // being rescheduled during this long-running operation

        const clusterIDs = await stores.kotsAppStore.listClusterIDsForApp(appId);
        if (clusterIDs.length === 0) {
          throw new Error("no clusters to transition for application");
        }
        sendInitialGitCommitsForAppDownstream(stores, appId, clusterIDs[0]);

        return true;
      } catch (err) {
        console.log(err);
        const gitOpsError = err.errno ? err.errno : "Unknown error connecting to repo";
        await stores.kotsAppStore.setGitOpsError(gitOpsCreds.id, gitOpsError);
        return false;
      } finally {
        // TODO delete
      }
    },

    async createKotsDownstream(root: any, args: any, context: Context) {
      context.requireSingleTenantSession();

      const { appId, clusterId } = args;

      const clusters = await stores.clusterStore.listAllUsersClusters();

      const cluster = _.find(clusters, (c: Cluster) => {
        return c.id === clusterId;
      });

      if (!cluster) {
        throw new ReplicatedError(`Cluster with the ID of ${clusterId} was either not found or you do not have permission to access it.`);
      }

      await stores.kotsAppStore.createDownstream(appId, cluster.title, clusterId);
      return true;
    },

    async uploadKotsLicense(root: any, args: any, context: Context) {
      try {
        context.requireSingleTenantSession();

        const { value } = args;
        const parsedLicense = yaml.safeLoad(value);

        const clusters = await stores.clusterStore.listAllUsersClusters();
        let downstream;
        for (const cluster of clusters) {
          if (cluster.title === process.env["AUTO_CREATE_CLUSTER_NAME"]) {
            downstream = cluster;
          }
        }
        const name = parsedLicense.spec.appSlug.replace("-", " ");
        const kotsApp = await kotsAppFromLicenseData(value, name, downstream.title, stores);

        // Carefully now, peek at registry credentials to see if we need to prompt for them
        let needsRegistry = true;
        try {
          const kc = new k8s.KubeConfig();
          kc.loadFromDefault();
          const k8sApi = kc.makeApiClient(k8s.CoreV1Api);
          const res = await k8sApi.readNamespacedSecret("registry-creds", "default");
          if (res && res.body && res.body.data && res.body.data[".dockerconfigjson"]) {
            needsRegistry = false;
          }

        } catch {
          /* no need to handle, rbac problem or not a path we can read registry */
        }

        return {
          hasPreflight: kotsApp.hasPreflight,
          isAirgap: parsedLicense.spec.isAirgapSupported,
          needsRegistry,
          slug: kotsApp.slug,
          isConfigurable: kotsApp.isAppConfigurable()
        }
      } catch(err) {
        throw new ReplicatedError(err.message);
      }
    },

    async updateRegistryDetails(root: any, args: any, context: Context) {
      context.requireSingleTenantSession();

      await stores.kotsAppStore.clearImageRewriteStatus();

      const { appSlug, hostname, username, password, namespace } = args.registryDetails;
      const appId = await stores.kotsAppStore.getIdFromSlug(appSlug);

      const downstreams = await stores.kotsAppStore.listDownstreamsForApp(appId);

      const currentSettings = await stores.kotsAppStore.getAppRegistryDetails(appId);
      if (currentSettings.registryHostname === hostname && currentSettings.namespace === namespace) {
        await stores.kotsAppStore.updateRegistryDetails(appId, hostname, username, password, namespace);
        return true;
      }

      await stores.kotsAppStore.setImageRewriteStatus("Updating registry settings", "running");

      const liveness = new Repeater(() => {
        return new Promise((resolve) => {
          stores.kotsAppStore.updateImageRewriteStatusLiveness().finally(() => {
            resolve();
          })
        });
      }, 1000);
      liveness.start();

      const updateFunc = async function(): Promise<void> {
        try {
          if (downstreams.length > 0) {
            const tmpDir = tmp.dirSync();
            try {
              const newVersionArchive = path.join(tmpDir.name, "archive.tar.gz");
              const app = await stores.kotsAppStore.getApp(appId);
              const registryInfo: KotsAppRegistryDetails = {
                registryHostname: hostname,
                registryUsername: username,
                registryPassword: password,
                registryPasswordEnc: "",
                lastSyncedAt: "",
                namespace: namespace,
              }
              await kotsRewriteImagesInVersion(app, downstreams, registryInfo, newVersionArchive, stores);
            } finally {
              tmpDir.removeCallback();
            }
          }
        } finally {
          liveness.stop();
        }
        await stores.kotsAppStore.updateRegistryDetails(appId, hostname, username, password, namespace);
      };

      updateFunc(); // rewrite images asyncronously

      return true;
    },

    async resumeInstallOnline(root: any, args: any, context: Context) {
      try {
        const { slug } = args;
        const appId = await stores.kotsAppStore.getIdFromSlug(slug);
        const app = await context.getApp(appId);
        const clusters = await stores.clusterStore.listAllUsersClusters();
        let downstream;
        for (const cluster of clusters) {
          if (cluster.title === process.env["AUTO_CREATE_CLUSTER_NAME"]) {
            downstream = cluster;
          }
        }
        const kotsApp = await kotsFinalizeApp(app, downstream.title, stores);
        await stores.kotsAppStore.setKotsAppInstallState(appId, "installed");
        return {
          ...kotsApp,
          isConfigurable: kotsApp.isAppConfigurable()
        };
      } catch(err) {
        throw new ReplicatedError(err.message);
      }
    },

    async deployKotsVersion(root: any, args: any, context: Context) {
      const { upstreamSlug, sequence, clusterSlug } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(upstreamSlug);
      const app = await context.getApp(appId);

      const clusterId = await stores.clusterStore.getIdFromSlug(clusterSlug);

      await stores.kotsAppStore.deployVersion(app.id, sequence, clusterId);
      return true;
    },

    async deleteKotsDownstream(root: any, args: any, context: Context) {
      const { slug, clusterId } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(slug);
      const app = await context.getApp(appId);
      await stores.kotsAppStore.deleteDownstream(app.id, clusterId);
      return true;
    },

    async deleteKotsApp(root: any, args: any, context: Context) {
      const { slug } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(slug);
      const app = await context.getApp(appId);
      await stores.kotsAppStore.deleteApp(app.id);
      return true;
    },

    async updateAppConfig(root: any, args: any, context: Context) {
      const { slug, sequence, configGroups, createNewVersion } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(slug);
      const app = await context.getApp(appId);
      await app.updateAppConfig(stores, slug, sequence, configGroups, createNewVersion);
      return true;
    },

    async updateKotsApp(root: any, args: any, context: Context): Promise<Boolean> {
      const app = await context.getApp(args.appId);
      await stores.kotsAppStore.updateApp(app.id, args.appName, args.iconUri);
      return true;
    },
  }
}
