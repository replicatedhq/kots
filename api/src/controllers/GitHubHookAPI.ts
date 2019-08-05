import Express, { response } from "express";
import { BodyParams, Controller, HeaderParams, Post, Req, Res } from "ts-express-decorators";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import fs from "fs";
import { trackNewGithubInstall } from "../util/analytics";
import uuid from "uuid";
import GitHubApi from "@octokit/rest";
import WebhooksApi from "@octokit/webhooks";
import jwt from "jsonwebtoken";
import { Cluster } from "../cluster";
import * as _ from "lodash";

interface ErrorResponse {
  error: {};
}

/**
 *  gets hooks from github
 */
@Controller("api/v1/hooks/github")
export class GitHubHookAPI {
  @Post("")
  async githubHook(
    @Res() response: Express.Response,
    @Req() request: Express.Request,
    @HeaderParams("x-github-event") eventType: string,
    @BodyParams("") body?: { action?: string }, // we're just gonna cast this later
  ): Promise<{} | ErrorResponse> {
    logger.info({msg: `received github hook for eventType ${eventType}`});

    switch (eventType) {
      case "pull_request": {
        await this.handlePullRequest(request, body as WebhooksApi.WebhookPayloadPullRequest);
        response.status(204);
        return {};
      }

      case "installation": {
        const params = await Params.getParams();
        await this.handleInstallation(body as WebhooksApi.WebhookPayloadInstallation, params, request);
        response.status(204);
        return {};
      }

      case "installation_repositories":
          const params = await Params.getParams();
          await this.handleInstallationRepositories(body as WebhooksApi.WebhookPayloadInstallationRepositories, params, request);
          response.status(204);
          return {};

      // ignored
      case "integration_installation":
      case "integration_installation_repositories":
        response.status(204);
        return {};

      // default is an error
      default: {
        logger.info({msg: `Unexpected event type in GitHub hook: ${eventType}`});
        response.status(204);
        return {};
      }
    }
  }

  private async handleInstallation(installationEvent: WebhooksApi.WebhookPayloadInstallation, params: Params, request: Express.Request): Promise<void> {
    const installationData = installationEvent.installation;
    const senderData = installationEvent.sender;
    let numberOfOrgMembers;
    if (installationEvent.action === "created") {
      if (installationData.account.type === "Organization") {
        const github = new GitHubApi();
        github.authenticate({
          type: "app",
          token: await getGitHubBearerToken()
        });

        const { data: { token } } = await github.apps.createInstallationToken({installation_id: installationData.id.toString()});
        github.authenticate({
          type: "token",
          token,
        });

        const org = installationData.account.login;
        const { data: membersData } = await github.orgs.getMembers({org});
        numberOfOrgMembers = membersData.length;

        for (const memberData of membersData) {
          await request.app.locals.stores.clusterStore.updateClusterGithubInstallationId(memberData.id, installationData.id);
        }
      } else {
        numberOfOrgMembers = 0;

        await request.app.locals.stores.clusterStore.updateClusterGithubInstallationId(installationData.account.id, installationData.id);
      };

      await request.app.locals.stores.githubInstall.createNewGithubInstall(installationData.id, installationData.account.login, installationData.account.type, numberOfOrgMembers, installationData.account.html_url, senderData.login);
      if (params.segmentioAnalyticsKey) {
        trackNewGithubInstall(params, uuid.v4(), "New Github Install", senderData.login, installationData.account.login, installationData.account.html_url);
      }
    } else if (installationEvent.action === "deleted") {
      // marking rows as deleted. currently this is only informative
      await request.app.locals.stores.clusterStore.updateClusterGithubInstallationsAsDeleted(installationData.id);

      // deleting from db when uninstall from GitHub
      await request.app.locals.stores.githubInstall.deleteGithubInstall(installationData.id);
      // Should we delete all pullrequest notifications?
    }
  }

  private async handleInstallationRepositories(event: WebhooksApi.WebhookPayloadInstallationRepositories, params: Params, request: Express.Request): Promise<void> {
    switch (event.action) {
      case "added":
        for (const repo of event.repositories_added) {
          await request.app.locals.stores.clusterStore.updateClusterGithubInstallationRepoAdded(event.installation.id, event.installation.account.login, repo.name);
        }

      case "removed":
        // we handle this in the sync-pr-status cron job
    }
  }

  private async handlePullRequest(request: Express.Request, pullRequestEvent: WebhooksApi.WebhookPayloadPullRequest): Promise<void> {
    const handledActions: string[] = ["opened", "closed", "reopened"];
    if (handledActions.indexOf(pullRequestEvent.action) === -1) {
      return;
    }

    let status = pullRequestEvent.action;
    if (pullRequestEvent.pull_request.merged) {
      status = "merged";
    }

    const owner = pullRequestEvent.pull_request.base.repo.owner.login;
    const repo = pullRequestEvent.pull_request.base.repo.name;

    const clusters = await request.app.locals.stores.clusterStore.listClustersForGitHubRepo(owner, repo);

    let commitShaFound = false;
    for (const cluster of clusters) {
      if (!cluster.gitOpsRef) {
        continue;
      }

      const watches = await request.app.locals.stores.watchStore.listForCluster(cluster.id!);
      for (const watch of watches) {
        try {
          const github = new GitHubApi();
          logger.debug({msg: "authenticating with bearer token"})
          github.authenticate({
            type: "app",
            token: await getGitHubBearerToken()
          });

          logger.debug({msg: "creating installation token for installationId", "installationId": cluster.gitOpsRef.installationId})
          const installationTokenResponse = await github.apps.createInstallationToken({installation_id: cluster.gitOpsRef.installationId});
          github.authenticate({
            type: "token",
            token: installationTokenResponse.data.token,
          });

          logger.debug({msg: "authenticated as app for installationId", "installationId": cluster.gitOpsRef.installationId});

          const params: GitHubApi.PullRequestsGetCommitsParams = {
            owner,
            repo,
            number: pullRequestEvent.number,
          };
          const getCommitsResponse = await github.pullRequests.getCommits(params);

          const sortedCommits = _.sortBy(getCommitsResponse.data, (commit) => {
            if (!commit.committed_at) {
              return undefined;
            }
            return new Date(commit.committed_at);
          });

          commitShaFound = sortedCommits.length > 0;

          for (const commit of sortedCommits) {
            const pendingVersion = await request.app.locals.stores.watchStore.getVersionForCommit(watch.id!, commit.sha);
            if (!pendingVersion) {
              continue;
            }
            await request.app.locals.stores.watchStore.updateVersionStatus(watch.id!, pendingVersion.sequence!, status);
            if (pullRequestEvent.pull_request.merged) {
              if (watch.currentVersion && pendingVersion.sequence! < watch.currentVersion.sequence!) {
                return;
              }

              const currentVersion = await request.app.locals.stores.watchStore.getCurrentVersion(watch.id!);
              if (currentVersion) {
                if (currentVersion.sequence > pendingVersion.sequence!) {
                  continue;
                }
              }

              await request.app.locals.stores.watchStore.setCurrentVersion(watch.id!, pendingVersion.sequence!, pullRequestEvent.pull_request.merged_at);
            }
          }
        } catch(err) {
          logger.info({msg: "could not update cluster because github said an error", err});
        }
      }
    }


    /////////
    // before adding commit_sha, some pending versions don't have them
    /////////
    if (!commitShaFound) {
      await handlePullRequestEventForVersionsWithoutCommitSha(clusters, request, pullRequestEvent, status);
    }
    ////////
    // end of pre-commit-sha compatibility code
    ////////
  }
}

export async function getGitHubBearerToken(): Promise<string> {
  const shipParams = await Params.getParams();

  let privateKey = await shipParams.githubPrivateKeyContents;
  if (!privateKey) {
    const filename = shipParams.githubPrivateKeyFile;
    logger.debug({msg: "using github private key from file"}, filename);
    privateKey = fs.readFileSync(filename).toString();
  } else {
    if (!privateKey.endsWith("\n")) {
      privateKey = `${privateKey}\n`;
    }
    logger.debug({msg: "using github private key from contents", size: privateKey.length})
  }

  const now = Math.floor(Date.now() / 1000)
  const payload = {
    iat: now,
    exp: now + 60,
    iss: shipParams.githubIntegrationID,
  }

  logger.debug({msg: "signing github jwt with payload", payload, "startOfPrivateKey": privateKey.substr(0, 100), "endOfPrivateKey": privateKey.substr(privateKey.length - 100)});
  const bearer = jwt.sign(payload, privateKey, {algorithm: "RS256"});
  return bearer;
}

async function handlePullRequestEventForVersionsWithoutCommitSha(clusters: Cluster[], request: Express.Request, pullRequestEvent: WebhooksApi.WebhookPayloadPullRequest, status: string) {
  for (const cluster of clusters) {
    const watches = await request.app.locals.stores.watchStore.listForCluster(cluster.id!);
    for (const watch of watches) {
      const pendingVersions = await request.app.locals.stores.watchStore.listPendingVersions(watch.id!);
      for (const pendingVersion of pendingVersions) {
        if (pendingVersion.pullrequestNumber === pullRequestEvent.number) {
          await request.app.locals.stores.watchStore.updateVersionStatus(watch.id!, pendingVersion.sequence!, status);
          if (pullRequestEvent.pull_request.merged) {
            // When a pull request closes multiple commits, the order in which hooks come in is random.
            // We should not update the current ssequence to something lower than what it already is.
            // This will create a bug where we show a PR as not merged but GH will show it as merged
            // because they automatically do it. This will be fixed when we verify commit sha's on our end.
            if (watch.currentVersion && pendingVersion.sequence! < watch.currentVersion.sequence!) {
              return;
            }
            await request.app.locals.stores.watchStore.setCurrentVersion(watch.id!, pendingVersion.sequence!, pullRequestEvent.pull_request.merged_at);
          }
          return;
        }
      }

      const pastVersions = await request.app.locals.stores.watchStore.listPastVersions(watch.id!);
      for (const pastVersion of pastVersions) {
        if (pastVersion.pullrequestNumber === pullRequestEvent.number) {
          await request.app.locals.stores.watchStore.updateVersionStatus(watch.id!, pastVersion.sequence!, status);
          if (pullRequestEvent.pull_request.merged) {
            await request.app.locals.stores.watchStore.setCurrentVersion(watch.id!, pastVersion.sequence!, pullRequestEvent.pull_request.merged_at);
          }
          return;
        }
      }

    }
  }
}
