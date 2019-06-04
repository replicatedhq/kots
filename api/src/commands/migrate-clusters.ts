import * as util from "util";
import { getPostgresPool } from "../util/persistence/db";
import { WatchStore } from "../watch/watch_store";
import { Params } from "../server/params";
import { tracer } from "../server/tracing";
import { NotificationStore } from "../notification/store";
import { ClusterStore, Cluster } from "../cluster";
import * as _ from "lodash";
import { UserStore } from "../user";

export const name = "migrate";
export const describe = "Migrate notifications to clusters";
export const builder = {

};

export const handler = async (argv) => {
  main(argv).catch((err) => {
    console.log(`Failed with error ${util.inspect(err)}`);
    process.exit(1);
  });
};

async function main(argv): Promise<any> {
  process.on('SIGTERM', function onSigterm () {
    process.exit();
  });

  const span = tracer().startSpan("migration.clusters");

  console.log(`Beginning migration to clusters...`);

  const pool = await getPostgresPool();
  const params = await Params.getParams();

  const watchStore = new WatchStore(pool, params);
  const notificationStore = new NotificationStore(pool, params);
  const clusterStore = new ClusterStore(pool, params);
  const userStore = new UserStore(pool);

  await userStore.migrateUsers();

  // const allWatches = await watchStore.listAllWatchesForAllTeams();
  // for (const watch of allWatches) {
  //   const userIds = await watchStore.listUsersForWatch(watch.id!);
  //   if (userIds.length === 0) {
  //     console.log(`no users for watch $${watch.watchName}, not migrating`);
  //     continue;
  //   }

  //   const owner = watch.slug!.split("/")[0];
  //   const firstUserId = userIds[0];
  //   userIds.shift();

  //   const parentState = JSON.parse(watch.stateJSON!);
  //   const childState = JSON.parse(watch.stateJSON!)

  //   // Move patches to the child
  //   if (parentState.v1 && parentState.v1.kustomize) {
  //     delete parentState.v1.kustomize;
  //   }

  //   const metadata = parentState.v1 && parentState.v1.metadata ? parentState.v1.metadata : {};
  //   console.log(`creating new parent watch for ${watch.watchName}`);

  //   try {
  //     const parentWatch = await watchStore.createNewWatch(JSON.stringify(parentState, null, 2), owner, firstUserId, metadata);

  //     // Change child upstream to be parent
  //     if (childState.v1) {
  //       childState.v1.upstream = `ship://ship-cluster/${parentWatch.id}`;

  //       console.log(`updating child watch state for watch ${watch.watchName}`);
  //       await watchStore.updateStateJSON(watch.id!, JSON.stringify(childState, null, 2), metadata);

  //       await watchStore.setParent(watch.id!, parentWatch.id!);
  //     }
  //   } catch (err) {
  //     console.log(`FAILED to create new parent watch for ${watch.watchName}`);
  //     console.log(err);
  //   }

  //   const watchNotifications = await notificationStore.listNotificationsOLD(watch.id!);

  //   for (const watchNotification of watchNotifications) {
  //     if (watchNotification.pullRequest) {
  //       console.log(`migrating PR from watch ${watch.watchName}`);

  //       const installationId = await notificationStore.getInstallationIdForPullRequestNotification(span.context(), watchNotification.id!);

  //       // generate a cluster name, to be used in a basic de-dupe attempt
  //       const clusterName = `${watchNotification.pullRequest.org}-${watchNotification.pullRequest.repo}-${watchNotification.pullRequest.branch}`;
  //       console.log(`looking for a cluster named ${clusterName} to target`);

  //       const clusters = await clusterStore.listClusters(firstUserId);
  //       let cluster = _.find(clusters, (cluster: Cluster) => {
  //         return cluster.title === clusterName;
  //       });

  //       if (!cluster) {
  //         const branch = watchNotification.pullRequest.branch ? watchNotification.pullRequest.branch : "";
  //         console.log(`creating a new cluster`);
  //         cluster = await clusterStore.createNewCluster(firstUserId, false, clusterName, "gitops", watchNotification.pullRequest.org, watchNotification.pullRequest.repo, branch, installationId);
  //       }

  //       if (!cluster) {
  //         console.error(`unable to find cluster`);
  //         return;
  //       }

  //       for (const userId of userIds) {
  //         await clusterStore.addUserToCluster(cluster.id, userId);
  //       }

  //       const path = watchNotification.pullRequest.rootPath ? watchNotification.pullRequest.rootPath : undefined;
  //       await watchStore.setCluster(watch.id!, cluster!.id!, path);

  //       // migrate the history
  //       const pullrequestHistoryItems = await notificationStore.listPullRequestHistory(span.context(), watchNotification.id!);
  //       for(const item of pullrequestHistoryItems) {
  //         await watchStore.createWatchVersion(watch.id!, item.createdOn, item.title, item.status!, item.sourceBranch!, item.sequence!, item.number!);
  //       }
  //     }
  //   }
  // }

  span.finish();

  console.log(`Migration to clusters completed!`);

  process.exit(0);
}


