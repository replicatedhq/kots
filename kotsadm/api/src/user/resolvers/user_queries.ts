import { Stores } from "../../schema/stores";
import { Context } from "../../context";
import { ReplicatedError } from "../../server/errors";
import { kotsTestRegistryCredentials } from "../../kots_app/kots_ffi";

export function UserQueries(stores: Stores) {
  return {
    async validateRegistryInfo(root: any, {slug, endpoint, username, password, org}: any, context: Context): Promise<String> {
      if (password === stores.kotsAppStore.getPasswordMask()) {
        const appId = await stores.kotsAppStore.getIdFromSlug(slug);
        const details = await stores.kotsAppStore.getAppRegistryDetails(appId);
        password = details.registryPassword;
      }
      try {
        const errorText = await kotsTestRegistryCredentials(endpoint, username, password, org);
        return errorText;
      } catch (err) {
        throw new ReplicatedError(`Error: ${err}`);
      }
    },
  }
}


