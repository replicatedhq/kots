import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function KotsLicenseQueries(stores: Stores) {
  return {
    async getAppLicense(root: any, { appId }, context: Context) {
      const app = await context.getApp(appId);
      return await stores.kotsLicenseStore.getAppLicense(app.id);
    },
  };
}
