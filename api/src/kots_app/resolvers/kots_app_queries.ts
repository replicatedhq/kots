import _ from "lodash";
import { Stores } from "../../schema/stores";
import { Context } from "../../context";
import { ReplicatedError } from "../../server/errors";
import { KotsApp, KotsVersion, KotsAppMetadata, KotsAppRegistryDetails, KotsConfigGroup, KotsDownstreamOutput } from "../";
import { Cluster } from "../../cluster";
import { kotsAppGetBranding } from "../kots_ffi";
import yaml from "js-yaml";
import { logger } from "../../server/logger";

export function KotsQueries(stores: Stores) {
  return {
    async getKotsMetadata(): Promise<KotsAppMetadata|null> {
      try {
        const rawBranding = await kotsAppGetBranding();
        const parsedBranding = yaml.safeLoad(rawBranding);
        const namespace = process.env["POD_NAMESPACE"] || "";

        return {
          name: parsedBranding.spec.title,
          iconUri: parsedBranding.spec.icon,
          namespace: namespace,
          isKurlEnabled: process.env["ENABLE_KURL"] === "1",
        };
      } catch (err) {
        console.log(err);
        logger.error("[kotsAppGetBranding] - Unable to retrieve or parse branding information");
        return null;
      }
    },

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
      const app = await context.getApp(_id);
      const downstreams = await stores.clusterStore.listClustersForKotsApp(app.id);

      return app.toSchema(downstreams, stores);
    },

    async listDownstreamsForApp(root: any, args: any, context: Context): Promise<Cluster[]> {
      const { slug } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(slug);
      const app = await context.getApp(appId);

      const downstreams = await stores.clusterStore.listClustersForKotsApp(app.id);
      let results: Cluster[] = [];
      _.map(downstreams, (downstream) => {
        const kotsSchemaCluster = downstream.toKotsAppSchema(appId, stores);
        results.push(kotsSchemaCluster);
      });
      return results;
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

    async getKotsDownstreamHistory(root: any, args: any, context: Context): Promise<KotsVersion[]> {
      const idFromSlug = await stores.kotsAppStore.getIdFromSlug(args.upstreamSlug);
      const clusterId = await stores.clusterStore.getIdFromSlug(args.clusterSlug);
      const app = await context.getApp(idFromSlug);
      const current = await app.getCurrentVersion(clusterId, stores);
      const past = await app.getPastVersions(clusterId, stores);
      const pending = await app.getPendingVersions(clusterId, stores);

      let versions: KotsVersion[];
      if (current === undefined) {
        versions = pending.concat(past);
      } else {
        versions = pending.concat(Array.of(current), past);
      }

      return versions;
    },

    async getAppRegistryDetails(root: any, args: any, context: Context): Promise<KotsAppRegistryDetails | {}> {
      const appId = await stores.kotsAppStore.getIdFromSlug(args.slug);
      const app = await context.getApp(appId);
      const details = await stores.kotsAppStore.getAppRegistryDetails(app.id);
      if (!details.registryHostname) {
        return {}
      }
      return details;
    },

    // TODO: This code is currently duplicated between kots apps and wathes.
    // It should be refactored so that you can get a file tree/download files
    // by a id/sequence number regardless of the app type.
    async getKotsApplicationTree(root: any, args: any, context: Context): Promise<string> {
      const appId = await stores.kotsAppStore.getIdFromSlug(args.slug);
      const app = await context.getApp(appId);
      const tree = await app.generateFileTreeIndex(args.sequence);
      if (_.isEmpty(tree) || !tree[0].children) {
        throw new ReplicatedError(`Unable to get files for app with ID of ${app.id}`);
      }
      return JSON.stringify(tree);
    },

    async getKotsFiles(root: any, args: any, context: Context): Promise<string> {
      const appId = await stores.kotsAppStore.getIdFromSlug(args.slug);
      const app = await context.getApp(appId);
      const files = await app.getFiles(args.sequence, args.fileNames);
      const jsonFiles = JSON.stringify(files.files);
      if (jsonFiles.length >= 5000000) {
        throw new ReplicatedError(`File is too large, the maximum allowed length is 5000000 but found ${jsonFiles.length}`);
      }
      return jsonFiles;
    },

    async getKotsConfigGroups(root: any, args: any, context: Context): Promise<KotsConfigGroup[]> {
      const appId = await stores.kotsAppStore.getIdFromSlug(args.slug);
      const app = await context.getApp(appId);
      return await app.getConfigGroups(args.sequence);
    },

    async getAirgapInstallStatus(root: any, args: any, context: Context): Promise<{ currentMessage: string, installStatus: string}> {
      return await stores.kotsAppStore.getAirgapInstallStatus();
    },

    async getKotsDownstreamOutput(root: any, args: any, context: Context): Promise<KotsDownstreamOutput> {
      const appId = await stores.kotsAppStore.getIdFromSlug(args.appSlug);
      const app = await context.getApp(appId);
      const clusterId = await stores.clusterStore.getIdFromSlug(args.clusterSlug);
      return await stores.kotsAppStore.getDownstreamOutput(app.id, clusterId, args.sequence);
    },
  }
}
