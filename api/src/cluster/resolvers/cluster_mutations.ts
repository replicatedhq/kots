import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";
import { trackUserClusterCreated } from "../../util/analytics";

export function ClusterMutations(stores: Stores, params: Params) {
  return {
    async createGitOpsCluster(root: any, { title, installationId, gitOpsRef }: any, context: Context) {
      const result = await stores.clusterStore.createNewCluster(context.session.userId, false, title, "gitops", gitOpsRef.owner, gitOpsRef.repo, gitOpsRef.branch, installationId);
      if (params.segmentioAnalyticsKey) {
        trackUserClusterCreated(params, context.session.userId, "New Ship Cloud GitOps Cluster Created", gitOpsRef.owner);
      }
      return result;
    },

    async createShipOpsCluster(root: any, { title }: any, context: Context) {
      const result = await stores.clusterStore.createNewCluster(context.session.userId, false, title, "ship");
      const user = await stores.userStore.getUser(context.session.userId);
      if (params.segmentioAnalyticsKey) {
        trackUserClusterCreated(params, context.session.userId, "New Ship Cloud ShipOps Cluster Created", user.githubUser ? user.githubUser.login : "");
      }
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
