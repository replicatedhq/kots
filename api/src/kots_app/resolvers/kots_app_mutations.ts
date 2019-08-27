import _ from "lodash";
import { Context } from "../../context";
import yaml from "js-yaml";
import { Stores } from "../../schema/stores";
import { Cluster } from "../../cluster";
import { ReplicatedError } from "../../server/errors";
import { kotsAppFromLicenseData } from "../kots_ffi";
import { logger } from "../../server/logger";

export function KotsMutations(stores: Stores) {
  return {
    async checkForKotsUpdates(root: any, args: any, context: Context) {
      const { appId } = args;

      console.log(appId);

      return false;
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
      await kotsAppFromLicenseData(value, name, downstream.title, stores);
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
