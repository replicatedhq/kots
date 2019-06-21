import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { SupportBundle } from "../";

interface uploadSupportBundleResponse {
  uploadUri: string;
  supportBundle: SupportBundle;
}

export function TroubleshootMutations(stores: Stores) {
  return {
    async uploadSupportBundle(root: any, args: any, context: Context): Promise<uploadSupportBundleResponse> {
      const bundle = await stores.troubleshootStore.createSupportBundle("TODO", args.size, args.notes);
      const uploadUri = await stores.troubleshootStore.signSupportBundlePutRequest(bundle);

      return {
        uploadUri,
        supportBundle: bundle,
      };
    },

    async markSupportBundleUploaded(root: any, args: any, context: Context): Promise<SupportBundle> {
      // TODO: size?
      return await stores.troubleshootStore.markSupportBundleUploaded(args.id)
    },
  }
}
