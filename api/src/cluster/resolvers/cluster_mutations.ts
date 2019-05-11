import { tracer } from "../../server/tracing";
import { Context } from "../../context";

export function ClusterMutations(stores: any) {
  return {
    async createGitOpsCluster(root: any, { title, installationId, gitOpsRef }: any, context: Context) {
      const result = await stores.clusterStore.createNewCluster(context.session.userId, false, title, "gitops", gitOpsRef.owner, gitOpsRef.repo, gitOpsRef.branch, installationId)
      return result;
    },

    async createShipOpsCluster(root: any, { title }: any, context: Context) {
      const result = await stores.clusterStore.createNewCluster(context.session.userId, false, title, "ship");
      return result;
    },

    async updateCluster(root: any, { clusterId, clusterName, gitOpsRef }: any, context: Context) {
      await stores.clusterStore.updateCluster(context.session.userId, clusterId, clusterName, gitOpsRef);
      const updatedCluster = stores.clusterStore.getCluster(clusterId);

      return updatedCluster;
    },

    async deleteCluster(root: any, { clusterId }: any, context: Context) {
      await stores.clusterStore.deleteCluster(context.session.userId, clusterId);

      return true;
    },
  }
}
