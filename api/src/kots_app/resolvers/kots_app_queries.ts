import _ from "lodash";
import { Stores } from "../../schema/stores";
import { Context } from "../../context";
import { ReplicatedError } from "../../server/errors";
import { KotsApp, KotsVersion } from "../";
import { Cluster } from "../../cluster";

export function KotsQueries(stores: Stores) {
  return {

    async getKotsApp(root: any, args: any, context: Context): Promise<KotsApp> {
      const { slug, id } = args;
      if (!id && !slug) {
        throw new ReplicatedError("One of slug or id is required");
      }
      let _id;
      if (slug) {
        _id = await stores.kotsAppStore.getIdFromSlug(slug)
      } else {
        _id = id;
      }
      const app = await stores.kotsAppStore.getApp(_id);

      const downstreams = await stores.clusterStore.listClustersForKotsApp(app.id);

      return app.toSchema(downstreams);
    },

    async listDownstreamsForApp(root: any, args: any, context: Context): Promise<Cluster[]> {
      const { slug } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(slug);
      const results = await stores.clusterStore.listClustersForKotsApp(appId);
      return results;
    },

    async listPendingWatchVersions(root: any, args: any, context: Context): Promise<KotsVersion[]> {
      const { slug, clusterId } = args;
      const id = await stores.kotsAppStore.getIdFromSlug(slug);
      const app = await stores.kotsAppStore.getApp(id);
      const downstreams = await stores.clusterStore.listClustersForKotsApp(id);
      const cluster = _.find(downstreams, (d: Cluster) => {
        return d.id === clusterId
      })
      return app.getPendingVersions(cluster!.id, stores);
    },

    async listPastWatchVersions(root: any, args: any, context: Context): Promise<KotsVersion[]> {
      const { slug, clusterId } = args;
      const id = await stores.kotsAppStore.getIdFromSlug(slug);
      const app = await stores.kotsAppStore.getApp(id);
      const downstreams = await stores.clusterStore.listClustersForKotsApp(id);
      const cluster = _.find(downstreams, (d: Cluster) => {
        return d.id === clusterId
      })
      return app.getPastVersions(cluster!.id, stores);
    },

    async getCurrentWatchVersion(root: any, args: any, context: Context): Promise<KotsVersion|undefined> {
      const { slug, clusterId } = args;
      const id = await stores.kotsAppStore.getIdFromSlug(slug);
      const app = await stores.kotsAppStore.getApp(id);
      const downstreams = await stores.clusterStore.listClustersForKotsApp(id);
      const cluster = _.find(downstreams, (d: Cluster) => {
        return d.id === clusterId
      })
      return app.getCurrentVersion(cluster!.id, stores);
    },

    // TODO: This code is currently duplicated between kots apps and wathes.
    // It should be refactored so that you can get a file tree/download files
    // by a id/sequence number regardless of the app type.
    async getKotsApplicationTree(root: any, args: any, context: Context): Promise<string> {
      const appId = await stores.kotsAppStore.getIdFromSlug(args.slug);
      const app = await stores.kotsAppStore.getApp(appId); // TODO: Move to context?
      const tree = await app.generateFileTreeIndex(args.sequence);
      if (_.isEmpty(tree) || !tree[0].children) {
        throw new ReplicatedError(`Unable to get files for app with ID of ${app.id}`);
      }
      return JSON.stringify(tree);
    },

    async getKotsFiles(root: any, args: any, context: Context): Promise<string> {
      const appId = await stores.kotsAppStore.getIdFromSlug(args.slug);
      const app = await stores.kotsAppStore.getApp(appId); // TODO: Move to context?
      const files = await app.getFiles(args.sequence, args.fileNames);
      const jsonFiles = JSON.stringify(files.files);
      if (jsonFiles.length >= 5000000) {
        throw new ReplicatedError(`File is too large, the maximum allowed length is 5000000 but found ${jsonFiles.length}`);
      }
      return jsonFiles;
    }

  }
}
