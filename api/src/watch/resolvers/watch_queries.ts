import * as _ from "lodash";
import { Watch, Contributor, Version, VersionDetail } from "../";
import { ReplicatedError } from "../../server/errors";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function WatchQueries(stores: Stores) {
  return {
    async getWatchVersion(root: any, args: any): Promise<VersionDetail> {
      const watch = await stores.watchStore.getWatch(args.id);
      const versionItem = await stores.watchStore.getOneVersion(args.id, args.sequence!);
      const params = await stores.watchStore.getLatestGeneratedFileS3Params(watch!.id!, args.sequence!);
      const download = await stores.watchDownload.findDeploymentFile(params);

      const versionItemDetail = {
        ...versionItem,
        rendered: download.contents.toString("utf-8"),
      }

      return versionItemDetail;
    },

    async listWatches(root: any, args: any, context: Context): Promise<Watch[]> {
      const watches = await stores.watchStore.listWatches(context.session.userId);
      const result = watches.map(watch => watch.toSchema(root, stores, context));
      return result;
    },

    async searchWatches(root: any, args: any, context: Context): Promise<Watch[]> {
      const { watchName } = args;
      const watches = await stores.watchStore.searchWatches(context.session.userId, watchName);
      return watches.map(watch => watch.toSchema(root, stores, context));
    },

    async getWatch(root: any, args: any, context: Context): Promise<Watch> {
      const { slug, id } = args;
      if (!id && !slug) {
        throw new ReplicatedError("One of slug or id is required", "bad_request");
      }
      const result = await stores.watchStore.findUserWatch(context.session.userId, { slug: slug!, id: id! });
      return result.toSchema(root, stores, context);
    },

    async watchContributors(root: any, args: any, context: Context): Promise<Contributor[]> {
      const { id } = args;
      const watch: Watch = await stores.watchStore.findUserWatch(context.session.userId, { id });
      return watch.getContributors(stores);
    },

    async listPendingWatchVersions(root: any, { watchId }: any, context: Context): Promise<Version[]> {
      const watch: Watch = await stores.watchStore.findUserWatch(context.session.userId, { id: watchId });
      return watch.getPendingVersions(stores);
    },

    async listPastWatchVersions(root: any, { watchId }: any, context: Context): Promise<Version[]> {
      const watch: Watch = await stores.watchStore.findUserWatch(context.session.userId, { id: watchId });
      return watch.getPastVersions(stores);
    },

    async getCurrentWatchVersion(root: any, args: any, context: Context): Promise<Version|undefined> {
      const watch: Watch = await stores.watchStore.findUserWatch(context.session.userId, { id: args.watchId });
      return watch.getCurrentVersion(stores);
    }

  }
}
