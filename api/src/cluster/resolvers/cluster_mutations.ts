import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function ClusterMutations(stores: Stores) {
  return {
    async createGitOpsCluster(root: any, { title, installationId, gitOpsRef }: any, context: Context) {
      const result = await stores.clusterStore.createNewCluster(context.session.userId, false, title, "gitops", gitOpsRef.owner, gitOpsRef.repo, gitOpsRef.branch, installationId)
      return result;
    },

    async createShipOpsCluster(root: any, { title }: any, context: Context) {
      const result = await stores.clusterStore.createNewCluster(context.session.userId, false, title, "ship");
      return result;
    },

    async updateCluster(root: any, args: any, context: Context) {
      const cluster = await context.getCluster(args.clusterId);

      await stores.clusterStore.updateCluster(context.session.userId, cluster.id, args.clusterName, args.gitOpsRef);
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
