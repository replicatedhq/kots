import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { SupportBundle } from "../";


interface SupportBundleUpload {
  uploadUri: string;
  supportBundle: SupportBundle;
}

export function TroubleshootMutations(stores: Stores) {
  return {
    async uploadSupportBundle(root: any, { token, size, notes }, context: Context): Promise<SupportBundleUpload> {
      const watchId = await stores.troubleshootStore.getWatchIdFromToken(token);
      const bundle = await stores.troubleshootStore.createSupportBundle(watchId, size, notes);
      const uploadUri = await stores.troubleshootStore.signSupportBundlePutRequest(bundle);

      return {
        uploadUri,
        supportBundle: bundle,
      };
    },

    async markSupportBundleUploaded(root: any, { id }, context: Context): Promise<SupportBundle> {
      // TODO: size?
      // TODO: file tree
      // TODO: async analysis
      return await stores.troubleshootStore.markSupportBundleUploaded(id);
    },
  }
}
