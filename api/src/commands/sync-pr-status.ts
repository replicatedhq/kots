import util from "util";
import clc from "cli-color";
import { Pool } from "pg";
import * as _ from "lodash";

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

const statusText = clc.green;
const blueText = clc.blue.bold;
const errorText = clc.red;

async function main(argv): Promise<any> {
  process.on('SIGTERM', () => {
    process.exit();
  });

  console.log(statusText("Syncing status of PR's with GitHub"));

  const pool = await getPostgresPool();
  console.log(statusText("Getting versions with a status of pending or open... "));
  const versions = await getVersions(pool);
  // TODO: retry 404s

  if (versions.length === 0) {
    console.log(blueText(`No versions with a status of "pending" or "open" were found.`));
    process.exit(0);
  }

  const params = await Params.getParams();
  const watchStore = new WatchStore(pool, params);
  let changedVersions: number[] = [];
  let four04Versions: number[] = [];
  let noshaVersions: number[] = [];
  let four04Clusters = {};
  let isDeletedClusters = {};
  let i = 0;

  for (const version of versions) {
    await sleep(1000);

    console.log("Fetching github PR: " + blueText(`https://github.com/${version.clusterOwner}/${version.clusterRepo}/pull/${version.pullrequestNumber}`));

    const github = new GitHubApi();
    github.authenticate({
      type: "app",
      token: await getGitHubBearerToken(),
    });

    try {
      let installationToken: string;

      try {
        installationToken = await getInstallationToken(github, pool, version, argv.dryRun);
      } catch (error) {
        if (error.code !== 404) {
          throw error;
        }

        console.log("Github installation " + blueText(`${version.clusterInstallationId}`) + " got 404");
        isDeletedClusters[version.clusterId] = true;
        continue;
      }

      github.authenticate({
        type: "token",
        token: installationToken,
      });

      try {
        await getRepo(github, pool, version, argv.dryRun);
      } catch (error) {
        if (error.code !== 404) {
          throw error;
        }

        console.log("Github repo got 404: " + blueText(`https://github.com/${version.clusterOwner}/${version.clusterRepo}`));
        four04Clusters[version.clusterId] = true;
        continue;
      }

      let pr: GitHubApi.GetResponse;

      try {
        pr = await getPullRequest(github, pool, version, argv.dryRun);
      } catch (error) {
        if (error.code !== 404) {
          throw error;
        }

        console.log("Github PR got 404: " + blueText(`https://github.com/${version.clusterOwner}/${version.clusterRepo}/pull/${version.pullrequestNumber}`));
        four04Versions.push(i);
        continue;
      }

      if (!version.commitSha) {
        // migrate old watch version rows with no commit sha
        const commits = await getPRCommits(github, version);
        if (commits.length > 0) {
          const lastCommit = commits.slice(-1)[0];
          if (lastCommit.sha) {
            console.log("Update github PR " + blueText(`${pr.number}`) + " commit sha to " + blueText(`${lastCommit.sha}`) + ": " + blueText(`https://github.com/${version.clusterOwner}/${version.clusterRepo}/pull/${pr.number}`));
            if (!argv.dryRun) {
              await updateWatchVersionCommitSha(pool, version.watchId, version.sequence, lastCommit.sha);
            }
            noshaVersions.push(i);
          }
        }
      }

      console.log(statusText(`Successfully fetched PR: https://github.com/${version.clusterOwner}/${version.clusterRepo}/pull/${pr.number}`));
      if (pr.merged && pr.state === "closed") {
        // PR is merged according to GitHub
        if (version.pullrequestStatus !== "merged") {
          console.log("Github said PR " + blueText(`${pr.number}`) + " was " + blueText("merged") + " but we had it marked as " + blueText(`${version.pullrequestStatus}`) + " check the status on github: " + blueText(`https://github.com/${version.clusterOwner}/${version.clusterRepo}/pull/${pr.number}`));
          if (!argv.dryRun) {
            await watchStore.updateVersionStatus(version.watchId!, version.sequence!, "merged");
            await pool.query(`UPDATE watch_version SET last_synced_at = current_timestamp WHERE watch_id = $1 AND sequence = $2`, [version.watchId, version.sequence]);
            await checkVersion(watchStore, version, pr);
          }
          changedVersions.push(i);
        }
      } else if (pr.state === "closed") {
        // PR was closed without merging according to GitHub
        if (version.pullrequestStatus !== "closed") {
          console.log("Github said PR " + blueText(`${pr.number}`) + " is " + blueText(`${pr.state}`) + " but we had it marked as " + blueText(`${version.pullrequestStatus}`) + " check the status on github: " + blueText(`https://github.com/${version.clusterOwner}/${version.clusterRepo}/pull/${pr.number}`));
          if (!argv.dryRun) {
            await watchStore.updateVersionStatus(version.watchId!, version.sequence!, "closed");
            await pool.query(`UPDATE watch_version SET last_synced_at = current_timestamp WHERE watch_id = $1 AND sequence = $2`, [version.watchId, version.sequence]);
            await checkVersion(watchStore, version, pr);
          }
          changedVersions.push(i);
        }
      } else {
        // PR is open according to GitHub
        if (version.pullrequestStatus !== "opened") {
          console.log("Github said PR " + blueText(`${pr.number}`) + " is " + blueText(`${pr.state}`) + " but we had it marked as " + blueText(`${version.pullrequestStatus}`) + " check the status on github: " + blueText(`https://github.com/${version.clusterOwner}/${version.clusterRepo}/pull/${pr.number}`));
          if (!argv.dryRun) {
            await watchStore.updateVersionStatus(version.watchId!, version.sequence!, "opened");
            await pool.query(`UPDATE watch_version SET last_synced_at = current_timestamp WHERE watch_id = $1 AND sequence = $2`, [version.watchId, version.sequence]);
            await checkVersion(watchStore, version, pr);
          }
          changedVersions.push(i);
        }
      }
    } catch (error) {
      console.log(errorText(`Failed to update PR https://github.com/${version.clusterOwner}/${version.clusterRepo}/pull/${version.pullrequestNumber}: ${error}`));
      continue;
    }
    i++;
  }
  console.log(blueText(`Checked ${versions.length} ${versions.length === 1 ? "row" : "rows"}`));
  console.log(blueText(`  - ${argv.dryRun ? "Will make" : "Made"} changes to ${changedVersions.length} ${changedVersions.length === 1 ? "row" : "rows"}`));
  console.log(blueText(`  - ${argv.dryRun ? "Will set" : "Set"} ${four04Versions.length} ${four04Versions.length === 1 ? "row" : "rows"} to 404 status`));;
  console.log(blueText(`  - ${argv.dryRun ? "Will set" : "Set"} ${Object.keys(four04Clusters).length} ${Object.keys(four04Clusters).length === 1 ? "cluster" : "clusters"} to 404 status`));
  console.log(blueText(`  - ${argv.dryRun ? "Will set" : "Set"} ${Object.keys(isDeletedClusters).length} ${Object.keys(isDeletedClusters).length === 1 ? "cluster" : "clusters"} to is deleted status`));
  console.log(blueText(`  - ${argv.dryRun ? "Will set" : "Set"} ${noshaVersions.length} ${noshaVersions.length === 1 ? "row" : "rows"} commit shas`));;
  process.exit(0);
}

export interface Version {
  watchId: string;
  sequence: number;
  clusterId: string;
  clusterOwner: string;
  clusterRepo: string;
  clusterInstallationId: string;
  pullrequestNumber: number;
  pullrequestStatus: string;
  commitSha: string;
  lastSyncedAt: Date;
}

async function getVersions(pool: Pool): Promise<Version[]> {
  const result = await pool.query(
    `SELECT wv.watch_id, wv.sequence, wc.cluster_id, cg.owner, cg.repo, cg.installation_id, wv.pullrequest_number, wv.status, wv.commit_sha, wv.last_synced_at
      FROM watch_version wv
      INNER JOIN watch_cluster wc ON wv.watch_id = wc.watch_id
      INNER JOIN cluster_github cg ON wc.cluster_id = cg.cluster_id AND (cg.is_deleted = FALSE OR cg.is_deleted is NULL) AND (cg.is_404 = FALSE OR cg.is_404 is NULL)
      WHERE wv.status IN ('opened', 'pending') AND (wv.is_404 = FALSE OR wv.is_404 IS NULL)`
  );
  const versions: Version[] = [];
  for (const row of result.rows) {
    versions.push({
      watchId: row.watch_id,
      sequence: row.sequence,
      clusterId: row.cluster_id,
      clusterOwner: row.owner,
      clusterRepo: row.repo,
      clusterInstallationId: row.installation_id,
      pullrequestNumber: row.pullrequest_number,
      pullrequestStatus: row.status,
      commitSha: row.commit_sha,
      lastSyncedAt: row.last_synced_at,
    });
  }
  return versions;
}

async function getInstallationToken(github: GitHubApi, pool: Pool, version: Version, dryRun: boolean): Promise<string> {
  try {
    const response = await github.apps.createInstallationToken({installation_id: version.clusterInstallationId});
    return response.data.token;
  } catch (error) {
    if (error.code === 404) {
      if (!dryRun) {
        await updateClusterAsIsDeleted(pool, version.clusterInstallationId);
      }
    }
    throw error;
  }
}

async function getRepo(github: GitHubApi, pool: Pool, version: Version, dryRun: boolean): Promise<GitHubApi.GetResponse> {
  try {
    const response = await github.repos.get({
      owner: version.clusterOwner,
      repo: version.clusterRepo,
    });
    return response.data;
  } catch (error) {
    if (error.code === 404) {
      if (!dryRun) {
        await updateClusterAs404(pool, version.clusterInstallationId);
      }
    }
    throw error;
  }
}

async function getPullRequest(github: GitHubApi, pool: Pool, version: Version, dryRun: boolean): Promise<GitHubApi.GetResponse> {
  try {
    const response = await github.pullRequests.get({
      owner: version.clusterOwner,
      repo: version.clusterRepo,
      number: version.pullrequestNumber,
    });
    return response.data;
  } catch (error) {
    if (error.code === 404) {
      if (!dryRun) {
        await updateWatchVersionAs404(pool, version.watchId, version.sequence);
      }
    }
    throw error;
  }
}

async function checkVersion(watchStore: WatchStore, version: Version, pr: GitHubApi.GetResponse): Promise<void> {
  console.log("Checking current version for " + blueText(`${version.watchId}`) + " (PR: " + blueText(`#${pr.number}`) + ")");
  const watch = await watchStore.getWatch(version.watchId);
  if (watch.currentVersion && version.sequence! < watch.currentVersion.sequence!) {
    return;
  }

  const currentVersion = await watchStore.getCurrentVersion(watch.id!);
  if (currentVersion) {
    if (currentVersion.sequence > version.sequence!) {
      return;
    }
  }

  await watchStore.setCurrentVersion(watch.id!, version.sequence!, pr.merged_at || undefined);
  console.log(statusText(`Updated current version sequence for ${version.watchId} (PR: #${pr.number}) from ${watch.currentVersion ? watch.currentVersion.sequence : "NULL"} to ${version.sequence}`));
}

async function getPRCommits(github: GitHubApi, version: Version): Promise<GitHubApi.GetCommitsResponseItem[]> {
  const params: GitHubApi.PullRequestsGetCommitsParams = {
    owner: version.clusterOwner,
    repo: version.clusterRepo,
    number: version.pullrequestNumber,
  };
  const getCommitsResponse = await github.pullRequests.getCommits(params);
  return getCommitsResponse.data;
}

async function updateClusterAsIsDeleted(pool: Pool, installationId: string) {
  await pool.query(
    `UPDATE cluster_github SET is_deleted = TRUE WHERE installation_id = $1`,
    [installationId],
  );
}

async function updateClusterAs404(pool: Pool, installationId: string) {
  await pool.query(
    `UPDATE cluster_github SET is_404 = TRUE WHERE installation_id = $1`,
    [installationId],
  );
}

async function updateWatchVersionCommitSha(pool: Pool, watchId: string, sequence: number, sha: string) {
  await pool.query(
    `UPDATE watch_version SET commit_sha = $3 WHERE watch_id = $1 AND sequence = $2`,
    [watchId, sequence, sha],
  );
}

async function updateWatchVersionAs404(pool: Pool, watchId: string, sequence: number) {
  await pool.query(
    `UPDATE watch_version SET is_404 = TRUE WHERE watch_id = $1 AND sequence = $2`,
    [watchId, sequence],
  );
}

async function sleep(ms: number) {
  return new Promise(resolve => {
    setTimeout(resolve, ms);
  });
}
