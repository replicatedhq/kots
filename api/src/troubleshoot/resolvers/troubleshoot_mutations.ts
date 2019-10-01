import { Context } from "../../context";
import _ from "lodash";
import { Stores } from "../../schema/stores";
import { SupportBundle, SupportBundleUpload } from "../";
import { analyzeSupportBundle } from "../troubleshoot_ffi";
import { Params } from "../../server/params";

export function TroubleshootMutations(stores: Stores, params: Params) {
  return {
    async uploadSupportBundle(root: any, { watchId, size }, context: Context): Promise<SupportBundleUpload> {
      const bundle = await stores.troubleshootStore.createSupportBundle(watchId, size);

      return {
        uploadUri: `${params.apiAdvertiseEndpoint}/api/v1/troubleshoot/${watchId}/${bundle.id}`,
        supportBundle: bundle,
      };
    },

    async markSupportBundleUploaded(root: any, { id }, context: Context): Promise<SupportBundle> {
      const bundle = await stores.troubleshootStore.getSupportBundle(id);
      // TODO: size?

      // Set file tree index
      const dirTree = await bundle.generateFileTreeIndex();
      await stores.troubleshootStore.assignTreeIndex(bundle.id, JSON.stringify(dirTree));

      const uploadedBundle = await stores.troubleshootStore.markSupportBundleUploaded(id);

          // Analyze it
      await analyzeSupportBundle(id, stores);

      return uploadedBundle;
    },
  }
}
