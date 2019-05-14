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
      const w = new Watch();
      const result = watches.map(watch => w.toSchema(watch, root, stores, context));
      return result;
    },

    async searchWatches(root: any, args: any, context: Context): Promise<Watch[]> {
      const { watchName } = args;
      const watches = await stores.watchStore.searchWatches(context.session.userId, watchName);
      const w = new Watch();
      return watches.map(watch => w.toSchema(watch, root, stores, context));
    },

    async getWatch(root: any, args: any, context: Context): Promise<Watch> {
      const { slug, id } = args;
      if (!id && !slug) {
        throw new ReplicatedError("One of slug or id is required", "bad_request");
      }
      const result = await stores.watchStore.findUserWatch(context.session.userId, { slug: slug!, id: id! });
      const watch = new Watch();
      return watch.toSchema(result, root, stores, context);
    },

    async watchContributors(root: any, args: any, context: Context): Promise<Contributor[]> {
      const { id } = args;
      const w: Watch = await stores.watchStore.findUserWatch(context.session.userId, { id });
      const watch = new Watch();
      return watch.getContributors(w.id!, stores);
    },

    async listPendingWatchVersions(root: any, { watchId }: any, context: Context): Promise<Version[]> {
      const w: Watch = await stores.watchStore.findUserWatch(context.session.userId, { id: watchId });
      const watch = new Watch();
      return watch.getPendingVersions(w.id, stores);
    },

    async listPastWatchVersions(root: any, { watchId }: any, context: Context): Promise<Version[]> {
      const w: Watch = await stores.watchStore.findUserWatch(context.session.userId, { id: watchId });
      const watch = new Watch();
      return watch.getPastVersions(w.id, stores);
    },

    async getCurrentWatchVersion(root: any, args: any, context: Context): Promise<Version|undefined> {
      const w: Watch = await stores.watchStore.findUserWatch(context.session.userId, { id: args.watchId });
      const watch = new Watch();
      return watch.getCurrentVersion(w.id!, stores);
    }

  }
}
