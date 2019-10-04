import _ from "lodash";
import { Context } from "../../context";
import yaml from "js-yaml";
import { Stores } from "../../schema/stores";
import { Cluster } from "../../cluster";
import { ReplicatedError } from "../../server/errors";
import { kotsAppFromLicenseData, kotsFinalizeApp, kotsAppCheckForUpdate } from "../kots_ffi";
import { KotsApp } from "../kots_app";

export function KotsMutations(stores: Stores) {
  return {
    async checkForKotsUpdates(root: any, args: any, context: Context) {
      const { appId } = args;

      const app = await context.getApp(appId);
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

    async updateRegistryDetails(root: any, args: any, context) {
      const { appSlug, hostname, username, password, namespace } = args.registryDetails;
      const appId = await stores.kotsAppStore.getIdFromSlug(appSlug);
      // TODO: encrypt password before setting it to the DB
      await stores.kotsAppStore.updateRegistryDetails(appId, hostname, username, password, namespace);
      return true;
    },

    async resumeInstallOnline(root: any, args: any, context: Context): Promise<KotsApp> {
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
      return kotsApp;
    },

    async deployKotsVersion(root: any, args: any, context: Context) {
      const { upstreamSlug, sequence, clusterSlug } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(upstreamSlug);
      const clusterId = await stores.clusterStore.getIdFromSlug(clusterSlug);

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
    },

    async updateAppConfig(root: any, args: any, context: Context) {
      const { slug, sequence, configGroups } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(slug);
      const app = await context.getApp(appId);
      await app.updateAppConfig(stores, slug, sequence, configGroups);
      return true;
    },

    async updateKotsApp(root: any, args: any, context: Context): Promise<Boolean> {
      const app = await context.getApp(args.appId);
      await stores.kotsAppStore.updateApp(app.id, args.appName, args.iconUri);
      return true;
    },
  }
}
