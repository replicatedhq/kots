import * as util from "util";
import { getPostgresPool } from "../util/persistence/db";
import { WatchStore } from "../watch/watch_store";
import { Params } from "../server/params";
import { tracer } from "../server/tracing";
import { NotificationStore } from "../notification/store";
import { ClusterStore } from "../cluster/cluster_store";
import * as _ from "lodash";

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

  const allWatches = await watchStore.listAllWatchesForAllTeams(span.context());
  for (const watch of allWatches) {
    const userIds = await watchStore.listUsersForWatch(span.context(), watch.id!);
    if (userIds.length === 0) {
      console.log(`no users for watch $${watch.watchName}, not migrating`);
      continue;
    }

    const owner = watch.slug!.split("/")[0];
    const firstUserId = userIds[0];
    userIds.shift();

    const parentState = JSON.parse(watch.stateJSON!);
    const childState = JSON.parse(watch.stateJSON!)

    // Move patches to the child
    if (parentState.v1 && parentState.v1.kustomize) {
      delete parentState.v1.kustomize;
    }

    const metadata = parentState.v1 && parentState.v1.metadata ? parentState.v1.metadata : {};
    console.log(`creating new parent watch for ${watch.watchName}`);
    const parentWatch = await watchStore.createNewWatch(span.context(), JSON.stringify(parentState, null, 2), owner, firstUserId, metadata);

    // Change child upstream to be parent
    if (childState.v1) {
      childState.v1.upstream = `ship://ship-cluster/${parentWatch.id}`;

      console.log(`updating child watch state for watch ${watch.watchName}`);
      await watchStore.updateStateJSON(span.context(), watch.id!, JSON.stringify(childState, null, 2), metadata);

      await watchStore.setParent(span.context(), watch.id!, parentWatch.id!);
    }

    const watchNotifications = await notificationStore.listNotifications(span.context(), watch.id!);

    for (const watchNotification of watchNotifications) {
      if (watchNotification.pullRequest) {
        console.log(`migrating PR from watch ${watch.watchName}`);

        const installationId = await notificationStore.getInstallationIdForPullRequestNotification(span.context(), watchNotification.id!);

        // generate a cluster name, to be used in a basic de-dupe attempt
        const clusterName = `${watchNotification.pullRequest.org}-${watchNotification.pullRequest.repo}-${watchNotification.pullRequest.branch}`;
        console.log(`looking for a cluster named ${clusterName} to target`);

        const clusters = await clusterStore.listClusters(span.context(), firstUserId);
        let cluster = _.find(clusters, (cluster) => {
          return cluster.title === clusterName;
        });

        if (!cluster) {
          const branch = watchNotification.pullRequest.branch ? watchNotification.pullRequest.branch : "";
          console.log(`creating a new cluster`);
          cluster = await clusterStore.createNewCluster(span.context(), firstUserId, false, clusterName, "gitops", watchNotification.pullRequest.org, watchNotification.pullRequest.repo, branch, installationId);
        }


        for (const userId of userIds) {
          await clusterStore.addUserToCluster(span.context(), cluster.id!, userId);
        }

        const path = watchNotification.pullRequest.rootPath ? watchNotification.pullRequest.rootPath : undefined;
        await watchStore.setCluster(span.context(), watch.id!, cluster!.id!, path);

        // migrate the history
        const pullrequestHistoryItems = await notificationStore.listPullRequestHistory(span.context(), watchNotification.id!);
        for(const item of pullrequestHistoryItems) {
          await watchStore.createWatchVersion(span.context(), watch.id!, item.createdOn, item.title, item.status!, item.sourceBranch!, item.sequence!, item.number!);
        }
      }
    }
  }

  span.finish();

  console.log(`Migration to clusters completed!`);

  process.exit(0);
}



// Table "public.pullrequest_notification"
// Column         |            Type             | Collation | Nullable | Default
// ------------------------+-----------------------------+-----------+----------+---------
// notification_id        | text                        |           | not null |
// org                    | text                        |           | not null |
// repo                   | text                        |           | not null |
// branch                 | text                        |           |          |
// root_path              | text                        |           |          |
// created_at             | timestamp without time zone |           | not null |
// github_installation_id | integer



// shipcloud-# \d cluster_github
//                Table "public.cluster_github"
//      Column      |  Type   | Collation | Nullable | Default
// -----------------+---------+-----------+----------+---------
//  cluster_id      | text    |           | not null |
//  owner           | text    |           | not null |
//  repo            | text    |           | not null |
//  branch          | text    |           |          |
//  installation_id | integer |           | not null |
// Indexes:

// create table watch_version (
//   watch_id text not null,
//   created_at timestamp without time zone,
//   version_label text not null,
//   status text not null default 'unknown',
//   source_branch text null,
//   sequence integer default 0,
//   pullrequest_number integer null
// );
