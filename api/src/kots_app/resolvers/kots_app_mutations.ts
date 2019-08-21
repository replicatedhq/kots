import _ from "lodash";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { Cluster } from "../../cluster";
import { ReplicatedError } from "../../server/errors";

export function KotsMutations(stores: Stores) {
  return {
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

    async deleteKotsDownstream(root: any, args: any, context: Context) {
      const { slug, clusterId } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(slug);
      await stores.kotsAppStore.deleteDownstream(appId, clusterId);
      return true;
    }
  }
}
