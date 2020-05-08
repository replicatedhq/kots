import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";

export function ClusterMutations(stores: Stores, params: Params) {
  return {
    async updateCluster(root: any, args: any, context: Context) {
      const cluster = await context.getCluster(args.clusterId);

      await stores.clusterStore.updateCluster(context.session.userId, cluster.id, args.clusterName);
      const updatedCluster = await context.getCluster(args.clusterId);

      return updatedCluster;
    },

    async deleteCluster(root: any, args: any, context: Context) {
      const cluster = await context.getCluster(args.clusterId);
      await stores.clusterStore.deleteCluster(context.session.userId, cluster.id);

      return true;
    },
  }
}
