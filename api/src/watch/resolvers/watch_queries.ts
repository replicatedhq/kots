import _ from "lodash";
import { Watch, Contributor, Version, VersionDetail } from "../";
import { ReplicatedError } from "../../server/errors";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { version } from "bluebird";

export function WatchQueries(stores: Stores) {
  return {
    async getWatchVersion(root: any, args: any, context: Context): Promise<VersionDetail> {
      const watch = await context.getWatch(args.id);
      const versionItem = await stores.watchStore.getOneVersion(watch.id, args.sequence!);
      const params = await stores.watchStore.getLatestGeneratedFileS3Params(watch.id, args.sequence!);
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
      const watches = await stores.watchStore.searchWatches(context.session.userId, args.watchName);
      return watches.map(watch => watch.toSchema(root, stores, context));
    },

    async getWatch(root: any, args: any, context: Context): Promise<Watch> {
      const { slug, id } = args;
      if (!id && !slug) {
        throw new ReplicatedError("One of slug or id is required");
      }
      const result = await stores.watchStore.findUserWatch(context.session.userId, { slug: slug!, id: id! });
      return result.toSchema(root, stores, context);
    },

    async getParentWatch(root: any, args: any, context: Context): Promise<Watch> {
      const { id, slug } = args;
      if (!id && !slug) {
        throw new ReplicatedError("One of ID or Slug is required to find a parent watch");
      }
      let _id = id;
      if (slug) {
        _id = await stores.watchStore.getIdFromSlug(slug);
      }
      const parentId = await stores.watchStore.getParentWatchId(_id);
      const parentWatch = await context.getWatch(parentId);
      return parentWatch.toSchema(root, stores, context);
    },

    async watchContributors(root: any, args: any, context: Context): Promise<Contributor[]> {
      const watch = await context.getWatch(args.id);
      return watch.getContributors(stores);
    },

    async listPendingWatchVersions(root: any, args: any, context: Context): Promise<Version[]> {
      const watch = await context.getWatch(args.watchId);
      return watch.getPendingVersions(stores);
    },

    async listPastWatchVersions(root: any, args: any, context: Context): Promise<Version[]> {
      const watch = await context.getWatch(args.watchId);
      return watch.getPastVersions(stores);
    },

    async getCurrentWatchVersion(root: any, args: any, context: Context): Promise<Version|undefined> {
      const watch = await context.getWatch(args.watchId);
      return watch.getCurrentVersion(stores);
    },

    async getDownstreamHistory(root: any, args: any, context: Context): Promise<Version[]> {
      const idFromSlug = await stores.watchStore.getIdFromSlug(args.slug);
      const watch = await context.getWatch(idFromSlug);
      const current = await watch.getCurrentVersion(stores);
      const past = await watch.getPastVersions(stores);
      const pending = await watch.getPendingVersions(stores);

      let versions: Version[];
      if (current === undefined) {
        versions = pending.concat(past);
      } else {
        versions = pending.concat(Array.of(current), past);
      }

      return versions;
    },

    async getApplicationTree(root: any, args: any, context: Context): Promise<string> {
      const watchId = await stores.watchStore.getIdFromSlug(args.slug);
      const watch = await context.getWatch(watchId);
      const tree = await watch.generateFileTreeIndex(args.sequence);
      if (_.isEmpty(tree) || !tree[0].children) {
        throw new ReplicatedError(`Unable to get files for watch with ID of ${watch.id}`);
      }
      // return children so you don't start with the "out" dir as top level in UI
      return JSON.stringify(tree[0].children);
    },

    async getFiles(root: any, args: any, context: Context): Promise<string> {
      const watchId = await stores.watchStore.getIdFromSlug(args.slug);
      const watch = await context.getWatch(watchId);
      const files = await watch.getFiles(args.sequence, args.fileNames);
      const jsonFiles = JSON.stringify(files.files);
      if (jsonFiles.length >= 5000000) {
        throw new ReplicatedError(`File is too large, the maximum allowed length is 5000000 but found ${jsonFiles.length}`);
      }
      return jsonFiles;
    }
  }
}
