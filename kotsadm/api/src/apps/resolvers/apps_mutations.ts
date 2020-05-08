import _ from "lodash";
import { generateKeyPairSync } from "crypto";
import sshpk from "sshpk";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { kotsEncryptString } from "../../kots_app/kots_ffi";
import { Params } from "../../server/params";

export function AppsMutations(stores: Stores) {
  return {
    async createGitOpsRepo(root: any, args: any, context: Context): Promise<boolean> {
      const { gitOpsInput } = args;

      const { publicKey, privateKey } = generateKeyPairSync("rsa", {
        modulusLength: 4096,
        publicKeyEncoding: {
          type: "pkcs1",
          format: "pem",
        },
        privateKeyEncoding: {
          type: "pkcs1",
          format: "pem",
        },
      });

      const params = await Params.getParams();
      const parsedPublic = sshpk.parseKey(publicKey, "pem");
      const sshPublishKey = parsedPublic.toString("ssh");

      const encryptedPrivateKey = await kotsEncryptString(params.apiEncryptionKey, privateKey);
      await stores.kotsAppStore.createGitOpsRepo(gitOpsInput.provider, gitOpsInput.uri, gitOpsInput.hostname, encryptedPrivateKey, sshPublishKey);

      return true;
    },

    async updateGitOpsRepo(root: any, args: any, context: Context): Promise<boolean> {
      const { gitOpsInput, uriToUpdate } = args;
      await stores.kotsAppStore.updateGitOpsRepo(uriToUpdate, gitOpsInput.uri, gitOpsInput.hostname);
      return true;
    },

    async resetGitOpsData(root: any, args: any, context: Context): Promise<boolean> {
      await stores.kotsAppStore.resetGitOpsData();
      return true;
    },
  }
}
