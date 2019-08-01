import util from "util";
import clc from "cli-color";

import { getPostgresPool } from "../util/persistence/db";
import GitHubApi from "@octokit/rest";
import { getGitHubBearerToken } from "../controllers/GitHubHookAPI";
import { Params } from "../server/params";
import { WatchStore } from "../watch/watch_store";

export const name = "sync-pr-status";
export const describe = "Sync status's of PR's with GitHub";
export const builder = {

};

export const handler = async (argv) => {
  main(argv).catch((err) => {
    console.log(`Failed with error ${util.inspect(err)}`);
    process.exit(1);
  });
};

const statusText = clc.xterm(78).bold;
const blueText = clc.xterm(75);

async function main(argv): Promise<any> {
  process.on('SIGTERM', () => {
    process.exit();
  });

  console.log(statusText("Syncing status of PR's with GitHub"));

  const pool = await getPostgresPool();
  console.log(statusText("Getting versions with a status of pending or open... "));
  const versions = await pool.query(
    `SELECT wv.watch_id, wv.pullrequest_number, wv.last_synced_at, wv.sequence, wv.status, wc.cluster_id, cg.owner, cg.repo, cg.installation_id
      FROM watch_version wv
      INNER JOIN watch_cluster wc ON wv.watch_id = wc.watch_id
      INNER JOIN cluster_github cg ON wc.cluster_id = cg.cluster_id
      WHERE status='opened' OR status='pending'`
  );

  if (versions.rowCount === 0) {
    console.log(blueText(`No versions with a status of "pending" or "open" were found.`));
    process.exit(0);
  } 

  const github = new GitHubApi();
  const params = await Params.getParams();
  const watchStore = new WatchStore(pool, params);
  let changedVersions: number[] = [];
  let i = 0;
  for (const version of versions.rows) {
    github.authenticate({
      type: "app",
      token: await getGitHubBearerToken()
    });

    try {
      const installationTokenResponse = await github.apps.createInstallationToken({installation_id: version.installation_id});
      github.authenticate({
        type: "token",
        token: installationTokenResponse.data.token,
      });
  
      const pr = await github.pullRequests.get({
        owner: version.owner,
        repo: version.repo,
        number: version.pullrequest_number
      });
      console.log(statusText(`successfully fetched pr ${version.owner}/${version.repo} #${version.pullrequest_number}`));
      if (pr.data.merged && pr.data.state === "closed") {
        // PR is merged according to GitHub
        if (version.status !== "merged") {
          if (!argv.dryRun) {
            console.log("Github said PR" + blueText(`${pr.data.number}`) + "was merged but we had it marked as" + blueText(`${version.status}`) + "check the status on github:" + blueText(`https://github.com/${version.owner}/${version.repo}/pull/${pr.data.number}`));
            await watchStore.updateVersionStatus(version.watch_id!, version.sequence!, "merged");
            await checkVersion(watchStore, version, pr.data);
          }
          changedVersions.push(i)
        }
      } else if (pr.data.state === "closed") {
        // PR was closed without merging according to GitHub
        if (version.status !== "closed") {
          if (!argv.dryRun) {
            console.log("Github said PR" + blueText(`${pr.data.number}`) + "is" + blueText(`${pr.data.state}`) + "but we had it marked as" + blueText(`${version.status}`) + "check the status on github:" + blueText(`https://github.com/${version.owner}/${version.repo}/pull/${pr.data.number}`));
            await watchStore.updateVersionStatus(version.watch_id!, version.sequence!, "closed");
            await checkVersion(watchStore, version, pr.data);
          }
          changedVersions.push(i)
        }
      } else {
        // PR is open according to GitHub
        if (version.status !== "opened") {
          if (!argv.dryRun) {
            console.log("Github said PR" + blueText(`${pr.data.number}`) + "is" + blueText(`${pr.data.state}`) + "but we had it marked as" + blueText(`${version.status}`) + "check the status on github:" + blueText(`https://github.com/${version.owner}/${version.repo}/pull/${pr.data.number}`));
            await watchStore.updateVersionStatus(version.watch_id!, version.sequence!, "opened");
            await checkVersion(watchStore, version, pr.data);
          }
          changedVersions.push(i)
        }
      }
    } catch (error) {
      console.log(statusText(`failed to fetch pr ${version.owner}/${version.repo} #${version.pullrequest_number}: ${error.code}`));
      await sleep(3000);
      continue;
    }
    i++
  }
  console.log(blueText(`Checked ${versions.rowCount} ${versions.rowCount === 1 ? "row" : "rows"} and ${argv.dryRun ? "will make" : "made"} changes to ${changedVersions.length} ${changedVersions.length === 1 ? "row" : "rows"}`));
  process.exit(0);
}

async function checkVersion(watchStore, version, pr) {
  console.log("Checking current version for" + clc.bold(`${version.watch_id}`) + "(PR: " + clc.bold(`#${pr.number}`) + ")");
  const watch = await watchStore.getWatch(version.watch_id);
  if (watch.currentVersion && version.sequence! < watch.currentVersion.sequence!) {
    return;
  }

  const currentVersion = await watchStore.getCurrentVersion(watch.id!);
  if (currentVersion) {
    if (currentVersion.sequence > version.sequence!) {
      return;
    }
  }

  await watchStore.setCurrentVersion(watch.id!, version.sequence!, pr.merged_at || null);
  console.log(statusText(`Updated current version sequence for ${version.watch_id} (PR: #${pr.number}) from ${watch.currentVersion ? watch.currentVersion.sequence : "NULL"} to ${version.sequence}`));
}

async function sleep(ms) {
  return new Promise(resolve => {
    setTimeout(resolve, ms);
  });
}