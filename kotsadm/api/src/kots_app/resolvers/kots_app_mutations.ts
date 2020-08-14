import _ from "lodash";
import { Context } from "../../context";
import path from "path";
import tmp from "tmp";
import yaml from "js-yaml";
import NodeGit from "nodegit";
import { Stores } from "../../schema/stores";
import { Cluster } from "../../cluster";
import { ReplicatedError } from "../../server/errors";
import {
  kotsDecryptString,
} from "../kots_ffi";
import { KotsApp } from "../kots_app"
import * as k8s from "@kubernetes/client-node";
import { Params } from "../../server/params";
import { Repeater } from "../../util/repeater";
import { sendInitialGitCommitsForAppDownstream } from "../gitops";
import { StatusServer } from "../../airgap/status";
import { logger } from "../../server/logger";

// tslint:disable-next-line max-func-body-length cyclomatic-complexity
export function KotsMutations(stores: Stores) {
  return {
    async updateAppGitOps(root: any, args: any, context: Context): Promise<boolean> {
      const { appId, clusterId, gitOpsInput } = args;
      const app = await context.getApp(appId);
      await stores.kotsAppStore.setDownstreamGitOps(app.id, clusterId, gitOpsInput.uri, gitOpsInput.branch, gitOpsInput.path, gitOpsInput.format, gitOpsInput.action);
      return true;
    },

    async disableAppGitops(root: any, args: any, context: Context): Promise<boolean> {
      const { appId, clusterId } = args;
      const app = await context.getApp(appId);
      const downstreamGitops = await stores.kotsAppStore.getDownstreamGitOps(app.id, clusterId);
      if (downstreamGitops.enabled) {
        await stores.kotsAppStore.disableDownstreamGitOps(appId, clusterId);
      }
      return true;
    },

    async ignorePreflightPermissionErrors(root: any, args: any, context: Context): Promise<boolean> {
      const { appSlug, clusterSlug, sequence } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(appSlug);
      const clusterId = await stores.clusterStore.getIdFromSlug(clusterSlug);
      await stores.kotsAppStore.ignorePreflightPermissionErrors(appId, clusterId, sequence);
      return true;
    },

    async retryPreflights(root: any, args: any, context: Context): Promise<boolean> {
      const { appSlug, clusterSlug, sequence } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(appSlug);
      const clusterId = await stores.clusterStore.getIdFromSlug(clusterSlug);
      await stores.kotsAppStore.retryPreflights(appId, clusterId, sequence);
      return true;
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

        await stores.kotsAppStore.setGitOpsError(appId, clusterId, "");
        // Send current and pending versions to git
        // We need a persistent, durable queue for this to handle the api container
        // being rescheduled during this long-running operation

        const clusterIDs = await stores.kotsAppStore.listClusterIDsForApp(appId);
        if (clusterIDs.length === 0) {
          throw new Error("no clusters to transition for application");
        }
        // NOTE (ethan): this is asyncronous but i didn't write the code
        sendInitialGitCommitsForAppDownstream(stores, appId, clusterIDs[0]);

        return true;
      } catch (err) {
        logger.error(err);
        const gitOpsError = err.errno ? `Error code ${err.errno}` : "Unknown error connecting to repo";
        await stores.kotsAppStore.setGitOpsError(appId, clusterId, gitOpsError);
        return false;
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

    async deployKotsVersion(root: any, args: any, context: Context) {
      const { upstreamSlug, sequence, clusterSlug } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(upstreamSlug);
      const app = await context.getApp(appId);

      await stores.kotsAppStore.deployVersion(app.id, sequence);
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

    async updateKotsApp(root: any, args: any, context: Context): Promise<Boolean> {
      const app = await context.getApp(args.appId);
      await stores.kotsAppStore.updateApp(app.id, args.appName, args.iconUri);
      return true;
    },

    async updateDownstreamsStatus(root: any, args: any, context: Context): Promise<Boolean> {
      const { slug, sequence, status } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(slug);
      const app = await context.getApp(appId);
      await stores.kotsAppStore.updateDownstreamsStatus(app.id, sequence, status, "");
      return true;
    },
  }
}

