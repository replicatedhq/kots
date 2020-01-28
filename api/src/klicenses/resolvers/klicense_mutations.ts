import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { syncLicense } from "../../controllers/kots/KotsAPI";
import { KLicense } from "../klicense";

export function KotsLicenseMutations(stores: Stores) {
  return {
    async syncAppLicense(root: any, args: any, context: Context): Promise<KLicense> {
      const { appSlug, airgapLicense } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(appSlug);
      const app = await context.getApp(appId)
      return await syncLicense(stores, app, airgapLicense);
    },
  }
}
