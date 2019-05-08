import { ClusterStore } from "./cluster_store";
import { SessionStore } from "../session/session_store";
import { Service } from "ts-express-decorators";
import { Query, Mutation } from "../schema/decorators";
import { ClusterItem } from "../generated/types";
import { tracer } from "../server/tracing";
import { Context } from "../context";
import { WatchStore } from "../watch/watch_store";

@Service()
export class Cluster {
  constructor(
    private readonly clusterStore: ClusterStore,
    private readonly watchStore: WatchStore,
    private readonly sessionStore: SessionStore,
  ) {}

  @Query("ship-cloud")
  async listClusters(root: any, args: any, context: Context): Promise<ClusterItem[]> {
    const span = tracer().startSpan("query.listClusters");

    const clusters = await this.clusterStore.listClusters(span.context(), context.session.userId);
    const result = clusters.map(cluster => this.toSchemaCluster(cluster, root, context));

    span.finish();

    return result;
  }

  @Query("ship-cloud")
  async getGitHubInstallationId(root: any, args: any, context: Context): Promise<any> {
    const gitSession = await this.sessionStore.getGithubSession(context.session.id);
    return gitSession.metadata;
  }

  @Mutation("ship-cloud")
  async createGitOpsCluster(root: any, { title, installationId, gitOpsRef }: any, context: Context): Promise<ClusterItem> {
    const span = tracer().startSpan("mutation.creaateGitOpsCluster");

    const result = await this.clusterStore.createNewCluster(span.context(), context.session.userId, false, title, "gitops", gitOpsRef.owner, gitOpsRef.repo, gitOpsRef.branch, installationId)

    span.finish();

    return result;
  }

  @Mutation("ship-cloud")
  async createShipOpsCluster(root: any, { title }: any, context: Context): Promise<ClusterItem> {
    const span = tracer().startSpan("mutation.createShipOpsCluster");

    const result = await this.clusterStore.createNewCluster(span.context(), context.session.userId, false, title, "ship");

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
  async updateCluster(root: any, { clusterId, clusterName, gitOpsRef }: any, context: Context): Promise<ClusterItem> {
    const span = tracer().startSpan("mutation.updateCluster");

    await this.clusterStore.updateCluster(span.context(), context.session.userId, clusterId, clusterName, gitOpsRef);
    const updatedCluster = this.clusterStore.getCluster(span.context(), clusterId);

    span.finish();

    return updatedCluster;
  }

  @Mutation("ship-cloud")
  async deleteCluster(root: any, { clusterId }: any, context: Context): Promise<boolean> {
    const span = tracer().startSpan("mutation.deleteCluster");

    await this.clusterStore.deleteCluster(span.context(), context.session.userId, clusterId);

    span.finish();

    return true;
  }
}
