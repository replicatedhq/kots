import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { ReplicatedError } from "../../server/errors";
import { getLatestLicense } from "../../kots_app/kots_ffi";
import yaml from "js-yaml";

export function KotsLicenseQueries(stores: Stores) {
  return {
    async getAppLicense(root: any, { appId }, context: Context) {
      const app = await context.getApp(appId);
      return await stores.kotsLicenseStore.getAppLicense(app.id);
    },

    async hasLicenseUpdates(root: any, args: any, context: Context) {
      const { appSlug } = args;
      const appId = await stores.kotsAppStore.getIdFromSlug(appSlug);
      const app = await context.getApp(appId);
      const license = await stores.kotsLicenseStore.getAppLicenseSpec(app.id);

      if (!license) {
        throw new ReplicatedError(`License not found for app with an ID of ${app.id}`);
      }
      
      const currentLicense = yaml.safeLoad(license);
      const latestLicenseYaml = await getLatestLicense(license);
      const latestLicense = yaml.safeLoad(latestLicenseYaml);

      return currentLicense.spec.licenseSequence !== latestLicense.spec.licenseSequence;
    },
  };
}
