import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import _ from "lodash";
import { ReplicatedError } from "../../server/errors";

export function TroubleshootQueries(stores: Stores) {
  return {
    async listKotsSupportBundles(root: any, { kotsSlug }, context: Context) {
      const kotsAppId = await stores.kotsAppStore.getIdFromSlug(kotsSlug);

      const supportBundles = await stores.troubleshootStore.listSupportBundles(kotsAppId);
      return _.map(supportBundles, async (supportBundle) => {
        return supportBundle.toSchema();
      });
    },

    async listSupportBundles(root: any, { watchSlug }, context: Context) {
      const appId = await stores.kotsAppStore.getIdFromSlug(watchSlug);
      const app = await context.getApp(appId);
      let supportBundles = await stores.troubleshootStore.listSupportBundles(app.id);

      return _.map(supportBundles, async (supportBundle) => {
        return supportBundle.toSchema();
      });
    },

    async getSupportBundle(root: any, { watchSlug }, context: Context) {
      const supportBundle = await stores.troubleshootStore.getSupportBundle(watchSlug);
      // TODO: Write some method that triest to fetch an app.
      // let app = await stores.kotsAppStore.getIdFromSlug(watchSlug);
      if (!supportBundle) {
        throw new ReplicatedError("not found");
      }

      return supportBundle.toSchema();
    },

    async supportBundleFiles(root, { bundleId, fileNames }, context: Context, { }): Promise<any> {
      const bundle = await stores.troubleshootStore.getSupportBundle(bundleId);
      const files = await bundle.getFiles(bundle, fileNames);
      const jsonFiles = JSON.stringify(files.files);
      if (jsonFiles.length >= 5000000) {
        throw new ReplicatedError(`File is too large, the maximum allowed length is 5000000 but found ${jsonFiles.length}`);
      }
      return jsonFiles;
    },

    async getSupportBundleCommand(root, { watchSlug }, context: Context): Promise<string> {
      if (!watchSlug) {
        return await stores.troubleshootStore.getSupportBundleCommand();
      }

      const appId = await stores.kotsAppStore.getIdFromSlug(watchSlug);
      const app = await context.getApp(appId);
      const bundleCommand = await stores.troubleshootStore.getSupportBundleCommand(app.slug);
      return bundleCommand;
    },
  };
}
