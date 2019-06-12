import { Validator } from "jsonschema";
import * as _ from "lodash";
import { ReplicatedError } from "../../server/errors";
import { logger } from "../../server/logger";
import { Context } from "../../context";
import { schema } from "../schema";
import { Stores } from "../../schema/stores";
import { Watch, Contributor } from "../";

export function WatchMutations(stores: Stores) {
  return {
    async deployWatchVersion(root: any, args: any, context: Context): Promise<boolean> {
      const watch = await context.getWatch(args.watchId);

      // TODO should probablly disallow this if it's a midtream or a gitops cluster?

      await stores.watchStore.setCurrentVersion(watch.id, args.sequence!);

      return true;
    },

    async updateStateJSON(root: any, args: any, context: Context): Promise<Watch> {

      const { slug, stateJSON } = args;

      let watch = await stores.watchStore.findUserWatch(context.session.userId, { slug: slug });

      validateJson(stateJSON, schema);

      const metadata = JSON.parse(stateJSON).v1.metadata;

      await stores.watchStore.updateStateJSON(watch.id!, stateJSON, metadata);

      watch = await stores.watchStore.getWatch(watch.id!);

      return watch;
    },

    async updateWatch(root: any, args: any, context: Context): Promise<Watch> {
      const watch = await context.getWatch(args.watchId);

      await stores.watchStore.updateWatch(watch.id, args.watchName, args.iconUri);
      const updatedWatch = await stores.watchStore.getWatch(watch.id);

      return updatedWatch.toSchema(root, stores, context);
    },

    async createWatch(root: any, { stateJSON, owner, clusterID, githubPath }: any, context: Context): Promise<Watch> {
      validateJson(stateJSON, schema);

      const metadata = JSON.parse(stateJSON).v1.metadata;
      const newWatch = await stores.watchStore.createNewWatch(stateJSON, owner, context.session.userId, metadata);

      return newWatch;
    },

    async deleteWatch(root: any, args: any, context: Context): Promise<boolean> {
      const watch = await context.getWatch(args.watchId);

      const notifications = await stores.notificationStore.listNotifications(watch.id);
      for (const notification of notifications) {
        await stores.notificationStore.deleteNotification(notification.id!);
      }

      // TODO delete from s3
      // They are still listed in ship_output_files, so we can reconcile this.

      await stores.watchStore.deleteWatch(watch.id);
      if (args.childWatchIds) {
        for (const id of args.childWatchIds) {
          if (id) {
            await stores.watchStore.deleteWatch(id);
          }
        }
      }

      return true;
    },

    async saveWatchContributors(root: any, args: any, context: Context): Promise<Contributor[]> {
      const watch = await context.getWatch(args.id);

      watch.addContributor(stores, context)

        // Remove existing contributors
        await stores.userStoreOld.removeExistingWatchContributorsExcept(watch.id, context.session.userId);

        // For each contributor, get user, if !user then create user
        for (const contributor of args.contributors) {
          const { githubId, login, avatar_url, email } = contributor!;

          let shipUser = await stores.userStore.tryGetGitHubUser(githubId!);
          if (!shipUser) {
            await stores.userStore.createGitHubUser(githubId!, login!, avatar_url!, email || "");
            await stores.userStore.createPasswordUser(githubId!, login!, "", "");
            shipUser = await stores.userStore.tryGetGitHubUser(githubId!);

            const allUsersClusters = await stores.clusterStore.listAllUsersClusters();
            for (const allUserCluster of allUsersClusters) {
              await stores.clusterStore.addUserToCluster(allUserCluster.id!, shipUser[0].id);
            }
          }
          // tslint:disable-next-line:curly
          if (shipUser && shipUser.id !== context.session.userId) await stores.userStoreOld.saveWatchContributor(shipUser.id, watch.id!);
        }

      return stores.watchStore.listWatchContributors(watch.id);
    },



  }
}

function validateJson(json, checkedSchema) {
  try {
    JSON.parse(json);
  } catch (e) {
    logger.info("JSON is not valid", e.message);
    throw new ReplicatedError("JSON is not valid");
  }

  const v = new Validator();
  const validationResult = v.validate(JSON.parse(json), schema);

  if (!validationResult.valid) {
    const resultErrors = validationResult.errors;
    const err = resultErrors.map(e => e.message)[0];
    const upperCaseErr = err.charAt(0).toUpperCase() + err.substr(1);
    throw new ReplicatedError(upperCaseErr);
  }
}
