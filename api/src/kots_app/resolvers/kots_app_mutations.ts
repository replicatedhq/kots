import _ from "lodash";
import { Context } from "../../context";
import yaml from "js-yaml";
import tmp from "tmp";
import path from "path";
import request from "request";
import { Stores } from "../../schema/stores";
import { Cluster } from "../../cluster";
import { ReplicatedError } from "../../server/errors";
import { kotsAppFromLicenseData, kotsAppCheckForUpdate, kotsAppFromAirgapData, kotsRewriteAndPushImageName } from "../kots_ffi";
import { extractFromURL, getImageFiles, pathToShortImageName, pathToImageName } from "../../airgap/airgap";

export function KotsMutations(stores: Stores) {
  return {
    async checkForKotsUpdates(root: any, args: any, context: Context) {
      const { appId } = args;

      const app = await stores.kotsAppStore.getApp(appId);
      const midstreamUpdateCursor = await stores.kotsAppStore.getMidstreamUpdateCursor(appId);

      const updateAvailable = await kotsAppCheckForUpdate(midstreamUpdateCursor, app, stores);

      return updateAvailable;
    },

    async createKotsDownstream(root: any, args: any, context: Context) {
      const { appId, clusterId } = args;

      const clusters = await stores.clusterStore.listAllUsersClusters();

      const cluster = _.find(clusters, (c: Cluster) => {
        return c.id === clusterId;
      });

      if (!cluster) {
        throw new ReplicatedError(`Cluster with the ID of ${clusterId} was either not found or you do not have permission to access it.`);
      }

      await stores.kotsAppStore.createDownstream(appId, cluster.title, clusterId);
      return true;
    },

    async uploadKotsLicense(root: any, args: any, context: Context) {
      const { value } = args;
      const parsedLicense = yaml.safeLoad(value);

      const clusters = await context.listClusters();
      let downstream;
      for (const cluster of clusters) {
        if (cluster.title === process.env["AUTO_CREATE_CLUSTER_NAME"]) {
          downstream = cluster;
        }
      }

      const name = parsedLicense.spec.appSlug.replace("-", " ")
      const kotsApp = await kotsAppFromLicenseData(value, name, downstream.title, stores);
      return kotsApp;
    },

    async getAirgapPutUrl(root: any, args: any, context: Context) {
      const { filename } = args;
      const url = await stores.kotsAppStore.getAirgapBundlePutUrl(filename);
      return url;
    },

    async markAirgapBundleUploaded(root: any, args: any, context: Context) {
      const dstDir = tmp.dirSync();

      try {

        const { filename, registryHost, registryNamespace, username, password } = args;

        // note we don't have to use signed url here... we can get a stream from s3 object.
        const url = await stores.kotsAppStore.getAirgapBundleGetUrl(filename);
        await extractFromURL(request({url: url}), dstDir.name);
        const imageFiles = await getImageFiles(path.join(dstDir.name, "images"));
        const imageMap = imageFiles.map(imageFile => {
          return {
            filePath: imageFile,
            shortName: pathToShortImageName(path.join(dstDir.name, "images"), imageFile),
            fullName: pathToImageName(path.join(dstDir.name, "images"), imageFile),
          }
        });
        for (const image of imageMap) {
          kotsRewriteAndPushImageName(image.filePath, image.shortName, registryHost, registryNamespace, username, password);
        }

        const app = await stores.kotsAppStore.getPendingKotsAirgapApp()

        const clusters = await stores.clusterStore.listAllUsersClusters();
        let downstream;
        for (const cluster of clusters) {
          if (cluster.title === process.env["AUTO_CREATE_CLUSTER_NAME"]) {
            downstream = cluster;
          }
        }

        await kotsAppFromAirgapData(app, String(app.license), dstDir.name, downstream.title, stores, registryHost, registryNamespace);

        await stores.kotsAppStore.updateRegistryDetails(app.id, registryHost, username, password, registryNamespace);

        return true;
      } finally {
        dstDir.removeCallback();
      }
    },

    async updateRegistryDetails(root: any, args: any, context) {
      const { appSlug, hostname, username, password, namespace } = args.registryDetails;
      const appId = await stores.kotsAppStore.getIdFromSlug(appSlug);
      // TODO: encrypt password before setting it to the DB
      await stores.kotsAppStore.updateRegistryDetails(appId, hostname, username, password, namespace);
      return true;
    },

    async deployKotsVersion(root: any, args: any, context: Context) {
      const { upstreamSlug, sequence, clusterSlug } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(upstreamSlug);
      const clusterId = await stores.clusterStore.getIdFromSlug(clusterSlug);

      await stores.kotsAppStore.deployVersion(appId, sequence, clusterId);
      return true;
    },

    async deleteKotsDownstream(root: any, args: any, context: Context) {
      const { slug, clusterId } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(slug);
      await stores.kotsAppStore.deleteDownstream(appId, clusterId);
      return true;
    },

    async deleteKotsApp(root: any, args: any, context: Context) {
      const { slug } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(slug);
      await stores.kotsAppStore.deleteApp(appId);
      return true;
    }

  }
}
