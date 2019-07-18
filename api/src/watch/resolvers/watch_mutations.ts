import { Validator } from "jsonschema";
import * as _ from "lodash";
import { ReplicatedError } from "../../server/errors";
import { logger } from "../../server/logger";
import { Context } from "../../context";
import { schema } from "../schema";
import { Stores } from "../../schema/stores";
import { Watch, Contributor } from "../";
import { Params } from "../../server/params";

export function WatchMutations(stores: Stores) {
  return {
    async deployWatchVersion(root: any, args: any, context: Context): Promise<boolean> {
      const watch = await context.getWatch(args.watchId);

      // TODO should probablly disallow this if it's a midtream or a gitops cluster?
      await stores.watchStore.setCurrentVersion(watch.id, args.sequence!, undefined, "deployed");

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

    async createWatch(root: any, { stateJSON }: any, context: Context): Promise<Watch> {
      validateJson(stateJSON, schema);

      const metadata = JSON.parse(stateJSON).v1.metadata;
      const newWatch = await stores.watchStore.createNewWatch(stateJSON, await context.getUsername(), context.session.userId, metadata);

      const editSession = await stores.editStore.createEditSession(context.session.userId, newWatch.id, true);

      const params = await Params.getParams();
      if (params.skipDeployToWorker) {
        return newWatch;
      }

      const deployedEditSession = await stores.editStore.deployEditSession(editSession.id);

      const now = new Date();
      const abortAfter = new Date(now.getTime() + (1000 * 60));
      while (new Date() < abortAfter) {
        const updatedEditSession = await stores.editStore.getSession(deployedEditSession.id);
        if (updatedEditSession.finishedOn) {
          return newWatch;
        }

        await sleep(1000);
      }

      throw new ReplicatedError("unable to create watch from state");
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

    async addWatchContributor(root: any, args: any, context: Context): Promise<Contributor[]> {
      const watch = await context.getWatch(args.watchId);

      let shipUser = await stores.userStore.tryGetGitHubUser(args.githubId);
      if (!shipUser) {
        await stores.userStore.createGitHubUser(args.githubId, args.login, args.avatarUrl, "");
        shipUser = await stores.userStore.tryGetGitHubUser(args.githubId);
      }

      if (!shipUser) {
        throw new ReplicatedError("Unknown user");
      }

      await stores.watchStore.addUserToWatch(watch.id, shipUser.id);

      const userIds = await stores.watchStore.listUsersForWatch(watch.id);
      const contributors = Promise.all(
        _.map(userIds, async (userId): Promise<Contributor> => {
          const contributor = await stores.userStore.getUser(userId);
          return {
            id: contributor.id,
            createdAt: contributor.createdAt.toString(),
            githubId: contributor.githubUser!.githubId,
            login: contributor.githubUser!.login,
            avatar_url: contributor.githubUser!.avatarUrl,
          };
        })
      );

      return contributors;
    },

    async removeWatchContributor(root: any, { watchId, contributorId }, context: Context): Promise<Contributor[]> {
      const watch = await context.getWatch(watchId);

      await stores.watchStore.removeUserFromWatch(watch.id, contributorId);

      const userIds = await stores.watchStore.listUsersForWatch(watch.id);
      const contributors = Promise.all(
        _.map(userIds, async (userId): Promise<Contributor> => {
          const contributor = await stores.userStore.getUser(userId);
          return {
            id: contributor.id,
            createdAt: contributor.createdAt.toString(),
            githubId: contributor.githubUser!.githubId,
            login: contributor.githubUser!.login,
            avatar_url: contributor.githubUser!.avatarUrl,
          };
        })
      );

      return contributors;
    },

    async checkForUpdates(root: any, { watchId }, context: Context): Promise<boolean> {
      const watch = await context.getWatch(watchId);

      await stores.watchStore.queueCheckForUpdates(watch.id);

      // we don't have a good way to determine if there are updates availabe...
      // so, we spin for a bit, check if there are pending versions, and return that

      // we need to get gql subscriptions hooked up so the worker can actually trigger events

      let pendingVersions = await stores.watchStore.listPendingVersions(watchId);
      if (pendingVersions.length > 0) {
        return true;
      }

      for (let i = 0; i < 15; i++) {
        await sleep(1000);
        pendingVersions = await stores.watchStore.listPendingVersions(watchId);
        if (pendingVersions.length > 0) {
          return true;
        }
      }

      return false;
    }
  }
}

function validateJson(json, checkedSchema) {
  try {
    JSON.parse(json);
  } catch (err) {
    logger.info({msg: "JSON is not valid", err});
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

function sleep(ms = 0) {
  return new Promise(r => setTimeout(r, ms));
}
