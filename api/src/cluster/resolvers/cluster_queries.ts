import { ClusterItem } from "../../generated/types";
import { tracer } from "../../server/tracing";
import { Context } from "../../context";
import * as _ from "lodash";
import { Stores } from "../../schema/stores";

export function ClusterQueries(stores: Stores) {
  return {
    async listClusters(root: any, args: any, context: Context): Promise<any[]> {
      const span = tracer().startSpan("query.listClusters");

      const clusters = await stores.clusterStore.listClusters(span.context(), context.session.userId);
      const result = _.map(clusters, async (cluster: ClusterItem) => {
        const applicationCount = await stores.clusterStore.getApplicationCount(cluster.id!);
        return toSchemaCluster(cluster, applicationCount);
      });

      span.finish();

      return result;
    },

    async getGitHubInstallationId(root: any, args: any, context: Context): Promise<any> {
      const gitSession = await stores.sessionStore.getGithubSession(context.session.id);
      return gitSession.metadata;
    }
  }
}

function toSchemaCluster(cluster: ClusterItem, applicationCount: number): any {
  return {
    ...cluster,
    totalApplicationCount: applicationCount,
  };
}
