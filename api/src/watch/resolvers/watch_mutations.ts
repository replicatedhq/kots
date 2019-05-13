
import { Validator } from "jsonschema";
import * as _ from "lodash";
import {
  ContributorItem,
  CreateWatchMutationArgs,
  DeleteWatchMutationArgs,
  SaveWatchContributorsMutationArgs,
  UpdateStateJsonMutationArgs,
  UpdateWatchMutationArgs,
  WatchItem,
  DeployWatchVersionMutationArgs,
} from "../../generated/types";
import { ReplicatedError } from "../../server/errors";
import { logger } from "../../server/logger";
import { Context } from "../../context";
import { tracer } from "../../server/tracing";
import { schema } from "../schema";
import { Stores } from "../../schema/stores";
import { toSchemaWatch } from "./watch_queries";

export function WatchMutations(stores: Stores) {
  return {
    async deployWatchVersion(root: any, args: DeployWatchVersionMutationArgs, context: Context): Promise<boolean> {
      const span = tracer().startSpan("mutation.deployShipOpsClusterVersion")

      const watch = await stores.watchStore.getWatch(args.watchId);

      // TODO should probablly disallow this if it's a midtream or a gitops cluster?

      await stores.watchStore.setCurrentVersion(watch.id!, args.sequence!);

      span.finish();

      return true;
    },

    async updateStateJSON(root: any, args: UpdateStateJsonMutationArgs, context: Context): Promise<WatchItem> {
      const span = tracer().startSpan("mutation.updateStateJSON");

      const { slug, stateJSON } = args;

      let watch = await stores.watchStore.findUserWatch(context.session.userId, { slug: slug });

      validateJson(stateJSON, schema);

      const metadata = JSON.parse(stateJSON).v1.metadata;

      await stores.watchStore.updateStateJSON(watch.id!, stateJSON, metadata);

      watch = await stores.watchStore.getWatch(watch.id!);

      span.finish();

      return watch;
    },

    async updateWatch(root: any, args: UpdateWatchMutationArgs, context: Context): Promise<WatchItem> {
      const span = tracer().startSpan("query.updateWatch");

      const { watchId, watchName, iconUri } = args;

      let watch = await stores.watchStore.findUserWatch(context.session.userId, { id: watchId });

      await stores.watchStore.updateWatch(watchId, watchName || undefined, iconUri || undefined);

      watch = await stores.watchStore.getWatch(watchId);

      span.finish();

      return toSchemaWatch(watch, root, context, stores);
    },

    async createWatch(root: any, { stateJSON, owner, clusterID, githubPath }: CreateWatchMutationArgs, context: Context): Promise<WatchItem> {
      const span = tracer().startSpan("mutation.createWatch");

      validateJson(stateJSON, schema);

      const metadata = JSON.parse(stateJSON).v1.metadata;

      const newWatch = await stores.watchStore.createNewWatch(stateJSON, owner, context.session.userId, metadata);
      span.finish();

      return newWatch;
    },

    async deleteWatch(root: any, { watchId, childWatchIds }: DeleteWatchMutationArgs, context: Context): Promise<boolean> {
      const span = tracer().startSpan("mutation.deleteWatch");

      const watch = await stores.watchStore.findUserWatch(context.session.userId, { id: watchId });

      const notifications = await stores.notificationStore.listNotifications(span.context(), watch.id!);
      for (const notification of notifications) {
        await stores.notificationStore.deleteNotification(span.context(), notification.id!);
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

      span.finish();

      return true;
    },

    async saveWatchContributors(root: any, args: SaveWatchContributorsMutationArgs, context: Context): Promise<ContributorItem[]> {
      const span = tracer().startSpan("mutation.saveWatchContributors");

      const { id, contributors } = args;
      const watch: WatchItem = await stores.watchStore.findUserWatch(context.session.userId, { id });


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
