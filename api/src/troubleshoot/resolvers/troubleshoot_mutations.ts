import { Context } from "../../context";
import _ from "lodash";
import { Stores } from "../../schema/stores";
import { SupportBundle, SupportBundleUpload } from "../";

export function TroubleshootMutations(stores: Stores) {
  return {
    async uploadSupportBundle(root: any, { watchId, size }, context: Context): Promise<SupportBundleUpload> {
      const bundle = await stores.troubleshootStore.createSupportBundle(watchId, size);
      const uploadUri = await stores.troubleshootStore.signSupportBundlePutRequest(bundle);

      return {
        uploadUri,
        supportBundle: bundle,
      };
    },

    async markSupportBundleUploaded(root: any, { id }, context: Context): Promise<SupportBundle> {
      const bundle = await stores.troubleshootStore.getSupportBundle(id);
      // TODO: size?
      
      // Set file tree index
      const dirTree = await bundle.generateFileTreeIndex();
      await stores.troubleshootStore.assignTreeIndex(bundle.id, JSON.stringify(dirTree));

      // TODO: async analysis
      return await stores.troubleshootStore.markSupportBundleUploaded(id);
    },
  }
}
