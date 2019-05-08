import { Validator } from "jsonschema";
import * as _ from "lodash";
import { Service } from "ts-express-decorators";
import { UserStore } from "../user/user_store";
import {
  ContributorItem,
  CreateWatchMutationArgs,
  DeleteWatchMutationArgs,
  GetWatchQueryArgs,
  SaveWatchContributorsMutationArgs,
  SearchWatchesQueryArgs,
  UpdateStateJsonMutationArgs,
  UpdateWatchMutationArgs,
  WatchContributorsQueryArgs,
  WatchItem,
  ClusterItem,
  VersionItem,
  ListPendingWatchVersionsQueryArgs,
  DeployWatchVersionMutationArgs,
  GetCurrentWatchVersionQueryArgs,
  VersionItemDetail,
  GetWatchVersionQueryArgs,
} from "../generated/types";
import { ShipNotification } from "../notification/resolver";
import { NotificationStore } from "../notification/store";
import { Mutation, Query } from "../schema/decorators";
import { ReplicatedError } from "../server/errors";
import { logger } from "../server/logger";
import { Context } from "../context";
import { tracer } from "../server/tracing";
import { schema } from "./schema";
import { WatchStore } from "./watch_store";
import { FeatureResolvers } from "../feature/resolver";
import { ClusterStore } from "../cluster/cluster_store";
import { WatchDownload } from "./download";

@Service()
export class Watch {
  constructor(
    private readonly watchStore: WatchStore,
    private readonly userStore: UserStore,
    private readonly clusterStore: ClusterStore,
    private readonly notificationStore: NotificationStore,
    private readonly shipNotificationResolver: ShipNotification,
    private readonly featureResolver: FeatureResolvers,
    private readonly downloadService: WatchDownload,
  ) {}

  @Mutation("ship-cloud")
  async deployWatchVersion(root: any, args: DeployWatchVersionMutationArgs, context: Context): Promise<boolean> {
    const span = tracer().startSpan("mutation.deployShipOpsClusterVersion")

    const watch = await this.watchStore.getWatch(span.context(), args.watchId);

    // TODO should probablly disallow this if it's a midtream or a gitops cluster?

    await this.watchStore.setCurrentVersion(span.context(), watch.id!, args.sequence!);

    span.finish();

    return true;
  }

  @Query("ship-cloud")
  async getWatchVersion(root: any, args: GetWatchVersionQueryArgs, context: Context): Promise<VersionItemDetail> {
    const span = tracer().startSpan("query.getWatchVersion");

    const watch = await this.watchStore.getWatch(span.context(), args.id);

    const versionItem = await this.watchStore.getOneVersion(span.context(), args.id, args.sequence!);
    const params = await this.watchStore.getLatestGeneratedFileS3Params(span.context(), watch!.id!, args.sequence!);
    const download = await this.downloadService.findDeploymentFile(span.context(), params);

    const versionItemDetail = {
      ...versionItem,
      rendered: download.contents.toString("utf-8"),
    }

    span.finish();

    return versionItemDetail;
  }

  @Query("ship-cloud")
  async listWatches(root: any, args: any, context: Context): Promise<WatchItem[]> {
    const span = tracer().startSpan("query.listWatches");

    const watches = await this.watchStore.listWatches(span.context(), context.session.userId);
    const result = watches.map(watch => this.toSchemaWatch(watch, root, context));

    span.finish();

    return result;
  }

  @Query("ship-cloud")
  async searchWatches(root: any, args: SearchWatchesQueryArgs, context: Context): Promise<WatchItem[]> {
    const span = tracer().startSpan("query.searchWatches");

    const { watchName } = args;

    const watches = await this.watchStore.searchWatches(span, context.session.userId, watchName);

    span.finish();

    return watches.map(watch => this.toSchemaWatch(watch, root, context));
  }

  @Query("ship-cloud")
  async getWatch(root: any, args: GetWatchQueryArgs, context: Context): Promise<WatchItem> {
    const span = tracer().startSpan("query.getWatch");

    const { slug, id } = args;
    if (!id && !slug) {
      throw new ReplicatedError("One of slug or id is required", "bad_request");
    }

    const result = await this.watchStore.findUserWatch(span.context(), context.session.userId, { slug: slug!, id: id! });

    span.finish();

    return this.toSchemaWatch(result, root, context);
  }

  @Mutation("ship-cloud")
  async updateStateJSON(root: any, args: UpdateStateJsonMutationArgs, context: Context): Promise<WatchItem> {
    const span = tracer().startSpan("mutation.updateStateJSON");

    const { slug, stateJSON } = args;

    let watch = await this.watchStore.findUserWatch(span.context(), context.session.userId, { slug: slug });

    this.validateJson(stateJSON, schema);

    const metadata = JSON.parse(stateJSON).v1.metadata;

    await this.watchStore.updateStateJSON(span.context(), watch.id!, stateJSON, metadata);

    watch = await this.watchStore.getWatch(span.context(), watch.id!);

    span.finish();

    return watch;
  }

  @Mutation("ship-cloud")
  async updateWatch(root: any, args: UpdateWatchMutationArgs, context: Context): Promise<WatchItem> {
    const span = tracer().startSpan("query.updateWatch");

    const { watchId, watchName, iconUri } = args;

    let watch = await this.watchStore.findUserWatch(span.context(), context.session.userId, { id: watchId });

    await this.watchStore.updateWatch(span.context(), watchId, watchName || undefined, iconUri || undefined);

    watch = await this.watchStore.getWatch(span.context(), watchId);

    span.finish();

    return watch;
  }

  @Mutation("ship-cloud")
  async createWatch(root: any, { stateJSON, owner, clusterID, githubPath }: CreateWatchMutationArgs, context: Context): Promise<WatchItem> {
    const span = tracer().startSpan("mutation.createWatch");

    this.validateJson(stateJSON, schema);

    const metadata = JSON.parse(stateJSON).v1.metadata;

    const newWatch = await this.watchStore.createNewWatch(span.context(), stateJSON, owner, context.session.userId, metadata);
    span.finish();

    return newWatch;
  }

  @Mutation("ship-cloud")
  async deleteWatch(root: any, { watchId, childWatchIds }: DeleteWatchMutationArgs, context: Context): Promise<boolean> {
    const span = tracer().startSpan("mutation.deleteWatch");

    const watch = await this.watchStore.findUserWatch(span.context(), context.session.userId, { id: watchId });

    const notifications = await this.notificationStore.listNotifications(span.context(), watch.id!);
    for (const notification of notifications) {
      await this.notificationStore.deleteNotification(span.context(), notification.id!);
    }

    // TODO delete from s3
    // They are still listed in ship_output_files, so we can reconcile this.

    await this.watchStore.deleteWatch(span.context(), watch.id!);
    if (childWatchIds) {
      for (const id of childWatchIds) {
        if (id) {
          await this.watchStore.deleteWatch(span.context(), id);
        }
      }
    }

    span.finish();

    return true;
  }

  @Query("ship-cloud")
  async watchContributors(root: any, args: WatchContributorsQueryArgs, context: Context): Promise<ContributorItem[]> {
    const span = tracer().startSpan("query.watchContributors");

    const { id } = args;

    const watch = await this.watchStore.findUserWatch(span.context(), context.session.userId, { id });

    return this.watchStore.listWatchContributors(span.context(), watch.id!);
  }

  @Mutation("ship-cloud")
  async saveWatchContributors(root: any, args: SaveWatchContributorsMutationArgs, context: Context): Promise<ContributorItem[]> {
    const span = tracer().startSpan("mutation.saveWatchContributors");

    const { id, contributors } = args;
    const watch: WatchItem = await this.watchStore.findUserWatch(span.context(), context.session.userId, { id });


    // await storeTransaction(this.userStore, async store => {
    //   // Remove existing contributors
    //   await store.removeExistingWatchContributorsExcept(span.context(), watch.id!, context.session.userId);

    //   // For each contributor, get user, if !user then create user
    //   for (const contributor of contributors) {
    //     const { githubId, login, avatar_url } = contributor!;

    //     let shipUser = await store.getUser(span.context(), githubId!);
    //     if (!shipUser.length) {
    //       await store.createGithubUser(span.context(), githubId!, login!, avatar_url!);
    //       await store.createShipUser(span.context(), githubId!, login!);
    //       shipUser = await store.getUser(span.context(), githubId!);

    //       const allUsersClusters = await this.clusterStore.listAllUsersClusters(span.context());
    //       for (const allUserCluster of allUsersClusters) {
    //         await this.clusterStore.addUserToCluster(span.context(), allUserCluster.id!, shipUser[0].id);
    //       }
    //     }
    //     // tslint:disable-next-line:curly
    //     if (shipUser[0].id !== context.session.userId) await store.saveWatchContributor(span.context(), shipUser[0].id, watch.id!);
    //   }
    // });

    return this.watchStore.listWatchContributors(span.context(), id);
  }

  @Query("ship-cloud")
  async listPendingWatchVersions(root: any, { watchId }: ListPendingWatchVersionsQueryArgs, context: Context): Promise<VersionItem[]> {
    const span = tracer().startSpan("query.listPendingWatchVersions");

    const watch: WatchItem = await this.watchStore.findUserWatch(span.context(), context.session.userId, { id: watchId });

    const pendingVersions = await this.watchStore.listPendingVersions(span.context(), watch.id!);

    span.finish();

    return pendingVersions;
  }

  @Query("ship-cloud")
  async listPastWatchVersions(root: any, { watchId }: ListPendingWatchVersionsQueryArgs, context: Context): Promise<VersionItem[]> {
    const span = tracer().startSpan("query.listPastWatchVersions");

    const watch: WatchItem = await this.watchStore.findUserWatch(span.context(), context.session.userId, { id: watchId });

    const pastVersions = await this.watchStore.listPastVersions(span.context(), watch.id!)

    span.finish();

    return pastVersions;
  }

  @Query("ship-cloud")
  async getCurrentWatchVersion(root: any, { watchId }: GetCurrentWatchVersionQueryArgs, context: Context): Promise<VersionItem|undefined> {
    const span = tracer().startSpan("query.getCurrentWatchVersion");

    const watch: WatchItem = await this.watchStore.findUserWatch(span.context(), context.session.userId, { id: watchId });

    const currentVersion = await this.watchStore.getCurrentVersion(span.context(), watch.id!)

    span.finish();

    return currentVersion;
  }

  async watchCluster(watchId: string): Promise<ClusterItem | void> {
    const span = tracer().startSpan("watchCluster");

    const cluster = await this.clusterStore.getForWatch(span.context(), watchId);

    span.finish();

    return cluster;  // TODO toSchemaCluster, but it's the same irght now
  }

  private toSchemaWatch(watch: WatchItem, root: any, ctx: Context): any {
    const schemaWatch = {...watch};
    schemaWatch.watches = schemaWatch.watches!.map(childWatch => this.toSchemaWatch(childWatch!, root, ctx));

    return {
      ...schemaWatch,
      cluster: async () => this.watchCluster(watch.id!),
      contributors: async () => this.watchContributors(root, { id: watch.id! }, ctx),
      notifications: async () => this.shipNotificationResolver.listNotifications(root, { watchId: watch.id! }, ctx),
      features: async () => this.featureResolver.watchFeatures(root, { watchId: watch.id! }, ctx),
      pendingVersions: async () => this.listPendingWatchVersions(root, { watchId: watch.id! }, ctx),
      pastVersions: async () => this.listPastWatchVersions(root, { watchId: watch.id! }, ctx),
      currentVersion: async () => this.getCurrentWatchVersion(root, { watchId: watch.id! }, ctx),
    };
  }

  private validateJson(json, checkedSchema) {
    try {
      JSON.parse(json);
    } catch (e) {
      logger.info("JSON is not valid", e.message);
      throw new ReplicatedError("JSON is not valid");
    }

    const v = new Validator();
    const validationResult = v.validate(JSON.parse(json), schema);

    if (!validationResult.valid) {
      const resultErrors = validationResult.errors;
      const err = resultErrors.map(e => e.message)[0];
      const upperCaseErr = err.charAt(0).toUpperCase() + err.substr(1);
      throw new ReplicatedError(upperCaseErr);
    }
  }


}
