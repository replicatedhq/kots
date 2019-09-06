import _ from "lodash";
import { Context } from "../../context";
import yaml from "js-yaml";
import fs from "fs";
import { Stores } from "../../schema/stores";
import { Cluster } from "../../cluster";
import { ReplicatedError } from "../../server/errors";
import { kotsAppFromLicenseData, kotsAppCheckForUpdate, kotsAppFromAirgapData } from "../kots_ffi";

export function KotsMutations(stores: Stores) {
  return {
    async checkForKotsUpdates(root: any, args: any, context: Context) {
      const { appId } = args;

      const app = await stores.kotsAppStore.getApp(appId);
      const midstreamUpdateCursor = await stores.kotsAppStore.getMidstreamUpdateCursor(appId);

      const updateAvailable = await kotsAppCheckForUpdate(midstreamUpdateCursor, app, stores);

      return updateAvailable;
    },

    async createKotsDownstream(root: any, args: any, context: Context) {
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
      const { value } = args;
      const parsedLicense = yaml.safeLoad(value);

      const clusters = await stores.clusterStore.listAllUsersClusters();
      let downstream;
      for (const cluster of clusters) {
        if (cluster.title === process.env["AUTO_CREATE_CLUSTER_NAME"]) {
          downstream = cluster;
        }
      }
      const name = parsedLicense.spec.appSlug.replace("-", " ")
      const kotsApp = await kotsAppFromLicenseData(value, name, downstream.title, stores);
      return kotsApp;
    },

    async getAirgapPutUrl(root: any, args: any, context: Context) {
      const { filename } = args;
      const url = await stores.kotsAppStore.getAirgapBundlePutUrl(filename);
      return url;
    },

    async markAirgapBundleUploaded(root: any, args: any, context: Context) {
      const { filename } = args;
      const app = await stores.kotsAppStore.getPendingKotsAirgapApp()

      const clusters = await stores.clusterStore.listAllUsersClusters();
      let downstream;
      for (const cluster of clusters) {
        if (cluster.title === process.env["AUTO_CREATE_CLUSTER_NAME"]) {
          downstream = cluster;
        }
      }

      const url = await stores.kotsAppStore.getAirgapBundleGetUrl(filename);

      await kotsAppFromAirgapData(app, String(app.license), url, downstream.title, stores);

      return true;
    },

    async updateRegistryDetails(root: any, args: any, context) {
      const { appSlug, hostname, username, password, namespace } = args.registryDetails;
      const appId = await stores.kotsAppStore.getIdFromSlug(appSlug);
      // TODO: encrypt password before setting it to the DB
      await stores.kotsAppStore.updateRegistryDetails(appId, hostname, username, password, namespace);
      return true;
    },

    async deployKotsVersion(root: any, args: any, context: Context) {
      const { upstreamSlug, sequence, clusterId } = args;
      let appId;
      if (!upstreamSlug) {
        appId = args.appId;
      } else {
        appId = await stores.kotsAppStore.getIdFromSlug(upstreamSlug);
      }

      if (!clusterId && !appId) {
        throw new ReplicatedError(`deployKotsVersion requires appId or upStreamSlug (appSlug). No appId found for upstreamSlug of ${upstreamSlug}`);
      }

      await stores.kotsAppStore.deployVersion(appId, sequence, clusterId);
      return true;
    },

    async deleteKotsDownstream(root: any, args: any, context: Context) {
      const { slug, clusterId } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(slug);
      await stores.kotsAppStore.deleteDownstream(appId, clusterId);
      return true;
    },

    async deleteKotsApp(root: any, args: any, context: Context) {
      const { slug } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(slug);
      await stores.kotsAppStore.deleteApp(appId);
      return true;
    }

  }
}
