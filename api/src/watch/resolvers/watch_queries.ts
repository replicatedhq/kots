import * as _ from "lodash";
import {
  ContributorItem,
  GetWatchQueryArgs,
  SearchWatchesQueryArgs,
  WatchContributorsQueryArgs,
  WatchItem,
  VersionItem,
  ListPendingWatchVersionsQueryArgs,
  GetCurrentWatchVersionQueryArgs,
  VersionItemDetail,
  GetWatchVersionQueryArgs,
} from "../../generated/types";
import { ReplicatedError } from "../../server/errors";
import { Context } from "../../context";
import { tracer } from "../../server/tracing";

export function WatchQueries(stores: any) {
  return {
    async getWatchVersion(root: any, args: GetWatchVersionQueryArgs, context: Context): Promise<VersionItemDetail> {
      const span = tracer().startSpan("query.getWatchVersion");

      const watch = await stores.watchStore.getWatch(span.context(), args.id);

      const versionItem = await stores.watchStore.getOneVersion(span.context(), args.id, args.sequence!);
      const params = await stores.watchStore.getLatestGeneratedFileS3Params(span.context(), watch!.id!, args.sequence!);
      const download = await this.downloadService.findDeploymentFile(span.context(), params);

      const versionItemDetail = {
        ...versionItem,
        rendered: download.contents.toString("utf-8"),
      }

      span.finish();

      return versionItemDetail;
    },

    async listWatches(root: any, args: any, context: Context): Promise<WatchItem[]> {
      const span = tracer().startSpan("query.listWatches");

      const watches = await stores.watchStore.listWatches(span.context(), context.session.userId);
      const result = watches.map(watch => toSchemaWatch(watch, root, context));

      span.finish();

      return result;
    },

    async searchWatches(root: any, args: SearchWatchesQueryArgs, context: Context): Promise<WatchItem[]> {
      const span = tracer().startSpan("query.searchWatches");

      const { watchName } = args;

      const watches = await stores.watchStore.searchWatches(span, context.session.userId, watchName);

      span.finish();

      return watches.map(watch => toSchemaWatch(watch, root, context));
    },

    async getWatch(root: any, args: GetWatchQueryArgs, context: Context): Promise<WatchItem> {
      const span = tracer().startSpan("query.getWatch");

      const { slug, id } = args;
      if (!id && !slug) {
        throw new ReplicatedError("One of slug or id is required", "bad_request");
      }

      const result = await stores.watchStore.findUserWatch(span.context(), context.session.userId, { slug: slug!, id: id! });

      span.finish();

      return toSchemaWatch(result, root, context);
    },

    async watchContributors(root: any, args: WatchContributorsQueryArgs, context: Context): Promise<ContributorItem[]> {
      const span = tracer().startSpan("query.watchContributors");

      const { id } = args;

      const watch = await stores.watchStore.findUserWatch(span.context(), context.session.userId, { id });

      return stores.watchStore.listWatchContributors(span.context(), watch.id!);
    },

    async listPendingWatchVersions(root: any, { watchId }: ListPendingWatchVersionsQueryArgs, context: Context): Promise<VersionItem[]> {
      const span = tracer().startSpan("query.listPendingWatchVersions");

      const watch: WatchItem = await stores.watchStore.findUserWatch(span.context(), context.session.userId, { id: watchId });

      const pendingVersions = await stores.watchStore.listPendingVersions(span.context(), watch.id!);

      span.finish();

      return pendingVersions;
    },

    async listPastWatchVersions(root: any, { watchId }: ListPendingWatchVersionsQueryArgs, context: Context): Promise<VersionItem[]> {
      const span = tracer().startSpan("query.listPastWatchVersions");

      const watch: WatchItem = await stores.watchStore.findUserWatch(span.context(), context.session.userId, { id: watchId });

      const pastVersions = await stores.watchStore.listPastVersions(span.context(), watch.id!)

      span.finish();

      return pastVersions;
    },

    async getCurrentWatchVersion(root: any, { watchId }: GetCurrentWatchVersionQueryArgs, context: Context): Promise<VersionItem|undefined> {
      const span = tracer().startSpan("query.getCurrentWatchVersion");

      const watch: WatchItem = await stores.watchStore.findUserWatch(span.context(), context.session.userId, { id: watchId });

      const currentVersion = await stores.watchStore.getCurrentVersion(span.context(), watch.id!)

      span.finish();

      return currentVersion;
    }

  }
}

function toSchemaWatch(watch: WatchItem, root: any, ctx: Context): any {
  const schemaWatch = {...watch};
  schemaWatch.watches = schemaWatch.watches!.map(childWatch => toSchemaWatch(childWatch!, root, ctx));

  return {
    ...schemaWatch,
  //   cluster: async () => this.watchCluster(watch.id!),
  //   contributors: async () => this.watchContributors(root, { id: watch.id! }, ctx),
  //   notifications: async () => this.shipNotificationResolver.listNotifications(root, { watchId: watch.id! }, ctx),
  //   features: async () => this.featureResolver.watchFeatures(root, { watchId: watch.id! }, ctx),
  //   pendingVersions: async () => this.listPendingWatchVersions(root, { watchId: watch.id! }, ctx),
  //   pastVersions: async () => this.listPastWatchVersions(root, { watchId: watch.id! }, ctx),
  //   currentVersion: async () => this.getCurrentWatchVersion(root, { watchId: watch.id! }, ctx),
  };
}
