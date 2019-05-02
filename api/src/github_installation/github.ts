import * as GitHubApi from "@octokit/rest";
import * as fs from "fs";
import * as jwt from "jsonwebtoken";
import { instrumented } from "monkit/dist";

import * as request from "request-promise";
import { StatusCodeError } from "request-promise/errors";
import { authorized } from "../auth/decorators";
import {
  GetBranchesResponseItem,
  GetForOrgResponse,
  GetInstallationsResponse,
  GetMembersResponseItem,
  GithubUser,
  InstallationOrganizationsQueryArgs,
  OrgMembersQueryArgs,
  OrgReposQueryArgs,
  RepoBranchesQueryArgs,
} from "../generated/types";
import { Query } from "../schema/decorators";
import { ReplicatedError } from "../server/errors";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import { Context } from "../server/server";

export class GithubAuthError extends Error {
  constructor(message?: string) {
    super(message || "Internal Server Error");
    this.name = "GithubAuthError";
  }
}

export class GitHub {
  constructor(private readonly accessTokensUrl: string, private readonly params: Params) {}

  @instrumented()
  async getToken(): Promise<string> {
    let key: string = "";
    if (this.params.githubPrivateKeyContents) {
      key = this.params.githubPrivateKeyContents;
    } else {
      key = fs.readFileSync(this.params.githubPrivateKeyFile).toString("utf-8");
    }

    const payload = {
      iss: this.params.githubIntegrationID,
      exp: Math.round(new Date().getTime() / 1000 + 60),
      iat: Math.round(new Date().getTime() / 1000),
    };

    const exchangeToken = jwt.sign(payload, key, { algorithm: "RS256" });

    const options = {
      method: "POST",
      uri: this.accessTokensUrl,
      headers: {
        "User-Agent": "Replicated",
        Accept: "application/vnd.github.machine-man-preview+json",
        Authorization: `Bearer ${exchangeToken}`,
      },
    };

    return request(options)
      .then((tokenResult: string) => (JSON.parse(tokenResult) as { token: string }).token)
      .catch(StatusCodeError, error => {
        const { statusCode } = error;
        if (statusCode === 400) {
          logger.info("Bad input provided to exchange token");
        }
        if (statusCode === 404) {
          throw new GithubAuthError();
        }
        throw error;
      });
  }

  @instrumented()
  @Query("ship-cloud")
  @authorized()
  // installationOrganizations returns organizations which have installed the ship github app
  async installationOrganizations(root: any, { page }: InstallationOrganizationsQueryArgs, context: Context): Promise<GetInstallationsResponse> {
    const github = new GitHubApi();
    github.authenticate({
      type: "token",
      token: context.auth,
    });

    let githubPage = 1;
    if (page) {
      githubPage = page;
    }

    const {
      data: installationData,
    }: {
      data: GitHubApi.GetInstallationsResponse;
    } = await github.users.getInstallations({
      page: githubPage,
    });

    const { total_count: totalCount, installations } = installationData as {
      total_count: number;
      installations: GitHubApi.GetInstallationsResponseInstallationsItem[];
    };

    return {
      totalCount,
      installations: installations.map(({ account }) => account),
    };
  }

  @instrumented()
  @Query("ship-cloud")
  @authorized()
  async orgRepos(root: any, { org, page }: OrgReposQueryArgs, context: Context): Promise<GetForOrgResponse> {
    const github = new GitHubApi({
      headers: {
        accept: "application/vnd.github.machine-man-preview+json",
      },
    });
    github.authenticate({
      type: "token",
      token: context.auth,
    });

    const lowerOrgName = org.toLowerCase();
    const installationId = context.metadata![lowerOrgName];
    if (!installationId) {
      throw new ReplicatedError("No matching installation id found");
    }

    let githubPage = 1;
    if (page) {
      githubPage = page;
    }

    const { data: installationData } = await github.users.getInstallationRepos({
      page: githubPage,
      installation_id: installationId,
    });

    return {
      repos: installationData.repositories,
      totalCount: installationData.total_count,
    };
  }

  @instrumented()
  @Query("ship-cloud")
  @authorized()
  async repoBranches(root: any, { owner, repo, page }: RepoBranchesQueryArgs, context: Context): Promise<GetBranchesResponseItem[]> {
    const github = new GitHubApi();
    github.authenticate({
      type: "token",
      token: context.auth,
    });

    let githubPage = 1;
    if (page) {
      githubPage = page;
    }

    const { data: repoBranches } = await github.repos.getBranches({
      owner,
      repo,
      page: githubPage,
    });

    return repoBranches;
  }

  @instrumented()
  @Query("ship-cloud")
  @authorized()
  async githubUser(root: any, args: any, context: Context): Promise<GithubUser> {
    const github = new GitHubApi();
    github.authenticate({
      type: "token",
      token: context.auth,
    });

    const { data: userData }: { data: GithubUser } = await github.users.get({});

    return userData;
  }

  @instrumented()
  @Query("ship-cloud")
  @authorized()
  async orgMembers(root: any, { org, page }: OrgMembersQueryArgs, context: Context): Promise<GetMembersResponseItem[]> {
    const github = new GitHubApi();
    github.authenticate({
      type: "token",
      token: context.auth,
    });

    let githubPage = 1;
    if (page) {
      githubPage = page;
    }

    const { data: membersData } = await github.orgs.getMembers({
      org,
      page: githubPage,
    });

    return membersData;
  }
}
