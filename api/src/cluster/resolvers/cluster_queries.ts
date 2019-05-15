import { Context } from "../../context";
import { Cluster } from "../";
import * as _ from "lodash";
import { Stores } from "../../schema/stores";

export function ClusterQueries(stores: Stores) {
  return {
    async listClusters(root: any, args: any, context: Context) {
      const clusters = await stores.clusterStore.listClusters(context.session.userId);
      const result = _.map(clusters, async (cluster: Cluster) => {
        const applicationCount = await stores.clusterStore.getApplicationCount(cluster.id!);
        return toSchemaCluster(cluster, applicationCount);
      });

      return result;
    },

    async getGitHubInstallationId(root: any, args: any, context: Context) {
      const gitSession = await stores.sessionStore.getGithubSession(context.session.id);
      return gitSession.metadata;
    }
  }
}

function toSchemaCluster(cluster: Cluster, applicationCount: number): any {
  return {
    ...cluster,
    totalApplicationCount: applicationCount,
  };
}
