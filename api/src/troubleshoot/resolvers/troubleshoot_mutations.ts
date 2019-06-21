import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { SupportBundle } from "../";

interface SupportBundleUpload {
  uploadUri: string;
  supportBundle: SupportBundle;
}

export function TroubleshootMutations(stores: Stores) {
  return {
    async uploadSupportBundle(root: any, args: any, context: Context): Promise<SupportBundleUpload> {
      const watchId = await stores.troubleshootStore.getWatchIdFromToken(args.token);
      const bundle = await stores.troubleshootStore.createSupportBundle(watchId, args.size, args.notes);
      const uploadUri = await stores.troubleshootStore.signSupportBundlePutRequest(bundle);

      return {
        uploadUri,
        supportBundle: bundle,
      };
    },

    async markSupportBundleUploaded(root: any, args: any, context: Context): Promise<SupportBundle> {
      // TODO: size?
      // TODO: async analysis
      return await stores.troubleshootStore.markSupportBundleUploaded(args.id)
    },
  }
}
