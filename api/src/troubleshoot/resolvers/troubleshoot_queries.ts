import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import * as _ from "lodash";
import { ReplicatedError } from "../../server/errors";

export function TroubleshootQueries(stores: Stores) {
  return {
    async watchCollectors(root: any, { watchId }, context: Context) {
      // watchCollectors is called by the support bundle container, and is not authenticated
      const collector = await stores.troubleshootStore.getPreferedWatchCollector(watchId);

      return {
        spec: collector.spec,
        hydrated: collector.spec,
      };
    },

    async listSupportBundles(root: any, { watchSlug }, context: Context) {
      const watch = await context.findWatch(watchSlug);
      const supportBundles = await stores.troubleshootStore.listSupportBundles(watch.id);
      return _.map(supportBundles, (supportBundle) => {
        return supportBundle.toSchema();
      });
    },

    async getSupportBundle(root: any, { watchSlug }, context: Context) {
      const supportBundle = await stores.troubleshootStore.getSupportBundle(watchSlug);
      const watch = context.getWatch(supportBundle.watchId);
      if (!watch) {
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

  };
}
