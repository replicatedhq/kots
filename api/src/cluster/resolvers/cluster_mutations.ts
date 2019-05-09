import { ClusterItem } from "../../generated/types";
import { tracer } from "../../server/tracing";
import { Context } from "../../context";

export function ClusterMutations(stores: any) {
  return {
    async createGitOpsCluster(root: any, { title, installationId, gitOpsRef }: any, context: Context): Promise<ClusterItem> {
      const span = tracer().startSpan("mutation.creaateGitOpsCluster");

      const result = await stores.clusterStore.createNewCluster(span.context(), context.session.userId, false, title, "gitops", gitOpsRef.owner, gitOpsRef.repo, gitOpsRef.branch, installationId)

      span.finish();

      return result;
    },

    async createShipOpsCluster(root: any, { title }: any, context: Context): Promise<ClusterItem> {
      const span = tracer().startSpan("mutation.createShipOpsCluster");

      const result = await stores.clusterStore.createNewCluster(span.context(), context.session.userId, false, title, "ship");

      span.finish();

      return result;
    },

    async updateCluster(root: any, { clusterId, clusterName, gitOpsRef }: any, context: Context): Promise<ClusterItem> {
      const span = tracer().startSpan("mutation.updateCluster");

      await stores.clusterStore.updateCluster(span.context(), context.session.userId, clusterId, clusterName, gitOpsRef);
      const updatedCluster = stores.clusterStore.getCluster(span.context(), clusterId);

      span.finish();

      return updatedCluster;
    },

    async deleteCluster(root: any, { clusterId }: any, context: Context): Promise<boolean> {
      const span = tracer().startSpan("mutation.deleteCluster");

      await stores.clusterStore.deleteCluster(span.context(), context.session.userId, clusterId);

      span.finish();

      return true;
    },
  }
}
