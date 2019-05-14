
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

      const watch = await stores.watchStore.getWatch(args.watchId);

      // TODO should probablly disallow this if it's a midtream or a gitops cluster?

      await stores.watchStore.setCurrentVersion(watch.id!, args.sequence!);

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
      const { watchId, watchName, iconUri } = args;

      let w = await stores.watchStore.findUserWatch(context.session.userId, { id: watchId });

      await stores.watchStore.updateWatch(watchId, watchName || undefined, iconUri || undefined);

      w = await stores.watchStore.getWatch(watchId);

      const watch = new Watch();

      return watch.toSchema(w, root, stores, context);
    },

    async createWatch(root: any, { stateJSON, owner, clusterID, githubPath }: any, context: Context): Promise<Watch> {

      validateJson(stateJSON, schema);

      const metadata = JSON.parse(stateJSON).v1.metadata;

      const newWatch = await stores.watchStore.createNewWatch(stateJSON, owner, context.session.userId, metadata);

      return newWatch;
    },

    async deleteWatch(root: any, { watchId, childWatchIds }: any, context: Context): Promise<boolean> {

      const watch = await stores.watchStore.findUserWatch(context.session.userId, { id: watchId });

      const notifications = await stores.notificationStore.listNotifications(watch.id!);
      for (const notification of notifications) {
        await stores.notificationStore.deleteNotification(notification.id!);
      }

      // TODO delete from s3
      // They are still listed in ship_output_files, so we can reconcile this.

      await stores.watchStore.deleteWatch(watch.id!);
      if (childWatchIds) {
        for (const id of childWatchIds) {
          if (id) {
            await stores.watchStore.deleteWatch(id);
          }
        }
      }

      return true;
    },

    async saveWatchContributors(root: any, args: any, context: Context): Promise<Contributor[]> {

      const { id, contributors } = args;
      const watch: Watch = await stores.watchStore.findUserWatch(context.session.userId, { id });


      // await storeTransaction(this.userStore, async store => {
      //   // Remove existing contributors
      //   await store.removeExistingWatchContributorsExcept(span.context(), watch.id!, context.session.userId);

      //   // For each contributor, get user, if !user then create user
      //   for (const contributor of contributors) {
      //     const { githubId, login, avatar_url } = contributor!;

      //     let shipUser = await store.getUser(span.context(), githubId!);
      //     if (!shipUser.length) {
      //       await store.createGithubUser(span.context(), githubId!, login!, avatar_url!);
      //       await store.createShipUser(span.context(), githubId!, login!);
      //       shipUser = await store.getUser(span.context(), githubId!);

      //       const allUsersClusters = await this.clusterStore.listAllUsersClusters(span.context());
      //       for (const allUserCluster of allUsersClusters) {
      //         await this.clusterStore.addUserToCluster(span.context(), allUserCluster.id!, shipUser[0].id);
      //       }
      //     }
      //     // tslint:disable-next-line:curly
      //     if (shipUser[0].id !== context.session.userId) await store.saveWatchContributor(span.context(), shipUser[0].id, watch.id!);
      //   }
      // });

      return stores.watchStore.listWatchContributors(id);
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
