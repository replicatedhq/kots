import { ClusterStore } from "./cluster_store";
import { SessionStore } from "../session/store";
import { Service } from "ts-express-decorators";
import { Query, Mutation } from "../schema/decorators";
import { authorized } from "../user/decorators";
import { ClusterItem } from "../generated/types";
import { tracer } from "../server/tracing";
import { Context } from "../server/server";
import { WatchStore } from "../watch/watch_store";

@Service()
export class Cluster {
  constructor(
    private readonly clusterStore: ClusterStore,
    private readonly watchStore: WatchStore,
    private readonly sessionStore: SessionStore,
  ) {}

  @Query("ship-cloud")
  @authorized()
  async listClusters(root: any, args: any, context: Context): Promise<ClusterItem[]> {
    const span = tracer().startSpan("query.listClusters");
    span.setTag("userId", context.userId);

    const clusters = await this.clusterStore.listClusters(span.context(), context.userId);
    const result = clusters.map(cluster => this.toSchemaCluster(cluster, root, context));

    span.finish();

    return result;
  }

  @Query("ship-cloud")
  @authorized()
  async getGitHubInstallationId(root: any, args: any, context: Context): Promise<any> {
    const span = tracer().startSpan("query.getGitHubInstallationId");
    span.setTag("context", context);

    const gitSession = await this.sessionStore.getGithubSession(span.context(), context.sessionId);
    span.finish();

    return gitSession.metadata;
  }

  @Mutation("ship-cloud")
  @authorized()
  async createGitOpsCluster(root: any, { title, installationId, gitOpsRef }: any, context: Context): Promise<ClusterItem> {
    const span = tracer().startSpan("mutation.creaateGitOpsCluster");

    const result = await this.clusterStore.createNewCluster(span.context(), context.userId, false, title, "gitops", gitOpsRef.owner, gitOpsRef.repo, gitOpsRef.branch, installationId)

    span.finish();

    return result;
  }

  @Mutation("ship-cloud")
  @authorized()
  async createShipOpsCluster(root: any, { title }: any, context: Context): Promise<ClusterItem> {
    const span = tracer().startSpan("mutation.createShipOpsCluster");

    const result = await this.clusterStore.createNewCluster(span.context(), context.userId, false, title, "ship");

    span.finish();

    return result;
  }

  async getApplicationCount(root: any, { clusterId }: any, context: Context): Promise<number> {
    const span = tracer().startSpan("mutation.getClusterApplicationCount");

    const result = await this.clusterStore.getApplicationCount(clusterId);

    span.finish();

    return result;
  }

  private toSchemaCluster(cluster: ClusterItem, root: any, ctx: Context): any {
    return {
      ...cluster,
      totalApplicationCount: async () => this.getApplicationCount(root, { clusterId: cluster.id! }, ctx)
    };
  }

  @Mutation("ship-cloud")
  @authorized()
  async updateCluster(root: any, { clusterId, clusterName, gitOpsRef }: any, context: Context): Promise<ClusterItem> {
    const span = tracer().startSpan("mutation.updateCluster");

    await this.clusterStore.updateCluster(span.context(), context.userId, clusterId, clusterName, gitOpsRef);
    const updatedCluster = this.clusterStore.getCluster(span.context(), clusterId);

    span.finish();

    return updatedCluster;
  }

  @Mutation("ship-cloud")
  @authorized()
  async deleteCluster(root: any, { clusterId }: any, context: Context): Promise<boolean> {
    const span = tracer().startSpan("mutation.deleteCluster");

    await this.clusterStore.deleteCluster(span.context(), context.userId, clusterId);

    span.finish();

    return true;
  }
}
