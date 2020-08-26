import _ from "lodash";
import { Stores } from "../../schema/stores";
import { Context } from "../../context";
import { ReplicatedError } from "../../server/errors";
import { KotsApp, KotsVersion, KotsAppRegistryDetails, KotsConfigGroup, KotsDownstreamOutput } from "../";
import { Cluster } from "../../cluster";
import { Params } from "../../server/params";

// tslint:disable-next-line max-func-body-length cyclomatic-complexity
export function KotsQueries(stores: Stores, params: Params) {
  return {
    async getKotsApp(root: any, args: any, context: Context): Promise<KotsApp> {
      const { slug, id } = args;
      if (!id && !slug) {
        throw new ReplicatedError("One of slug or id is required");
      }
      let _id: string;
      if (slug) {
        _id = await stores.kotsAppStore.getIdFromSlug(slug)
      } else {
        _id = id;
      }
      const app = await context.getApp(_id);
      const downstreams = await stores.clusterStore.listClustersForKotsApp(app.id);

      return app.toSchema(downstreams, stores);
    },

    async listPendingKotsVersions(root: any, args: any, context: Context): Promise<KotsVersion[]> {
      const { slug, clusterId } = args;
      const id = await stores.kotsAppStore.getIdFromSlug(slug);
      const app = await context.getApp(id);
      return app.getPendingVersions(clusterId, stores);
    },

    async listPastKotsVersions(root: any, args: any, context: Context): Promise<KotsVersion[]> {
      const { slug, clusterId } = args;
      const id = await stores.kotsAppStore.getIdFromSlug(slug);
      const app = await context.getApp(id);
      return app.getPastVersions(clusterId, stores);
    },

    async getCurrentKotsVersion(root: any, args: any, context: Context): Promise<KotsVersion|undefined> {
      const { slug, clusterId } = args;
      const id = await stores.kotsAppStore.getIdFromSlug(slug);
      const app = await context.getApp(id);
      return app.getCurrentVersion(clusterId, stores);
    },

    async getOnlineInstallStatus(root: any, args: any, context: Context): Promise<{ currentMessage: string, installStatus: string}> {
      return await stores.kotsAppStore.getOnlineInstallStatus();
    },

    async getAirgapInstallStatus(root: any, args: any, context: Context): Promise<{ currentMessage: string, installStatus: string}> {
      return await stores.kotsAppStore.getAirgapInstallStatus();
    },

    async getImageRewriteStatus(root: any, args: any, context: Context): Promise<{ currentMessage: string, status: string}> {
      return await stores.kotsAppStore.getImageRewriteStatus();
    },

    async getKotsDownstreamOutput(root: any, args: any, context: Context): Promise<KotsDownstreamOutput> {
      const appId = await stores.kotsAppStore.getIdFromSlug(args.appSlug);
      const app = await context.getApp(appId);
      const clusterId = await stores.clusterStore.getIdFromSlug(args.clusterSlug);
      return await stores.kotsAppStore.getDownstreamOutput(app.id, clusterId, args.sequence);
    },
  };
}
