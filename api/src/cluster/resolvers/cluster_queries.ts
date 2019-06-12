import { Context } from "../../context";
import { Cluster } from "../";
import * as _ from "lodash";
import { Stores } from "../../schema/stores";

export function ClusterQueries(stores: Stores) {
  return {
    async listClusters(root: any, args: any, context: Context) {
      const result = _.map(await context.listClusters(), async (cluster: Cluster) => {
        const applicationCount = await stores.clusterStore.getApplicationCount(cluster.id);
        return toSchemaCluster(cluster, applicationCount);
      });

      return result;
    },

    async getGitHubInstallationId(root: any, args: any, context: Context) {
      const gitSession = await stores.sessionStore.getGithubSession(context.session.sessionId);
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
