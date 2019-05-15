import * as _ from "lodash";

export class GitHubRef {
    id: string;
    installationId: number;
    repoFullName: string;
    branch: string;
    path: string;
    itemType: string;
    createdAt: Date;
  
    toSchema(): {} {
      return {
        id: this.id,
        owner: _.split(this.repoFullName, "/")[0],
        repoFullName: this.repoFullName,
        branch: this.branch,
        path: this.path,
      };
    }
  }
  
  export class GitHubRepo {
    name: string;
    fullName: string;
  
    toSchema(): {} {
      return {
        name: this.name,
        fullName: this.fullName,
      };
    }
  }
  
  export class GitHubInstallation {
    id: number;
    accessTokensUrl: string;
    accountLogin: string;
    accountId: number;
    accountType: string;
    createdAt: Date;
  
    toSchema(repos: GitHubRepo[]): {} {
      return {
        id: this.id,
        name: this.accountLogin,
        createdAt: this.createdAt,
        repos: repos.map((repo: GitHubRepo) => {
          return repo.toSchema();
        }),
      };
    }
  }
  
  export interface CreateGitHubInstallationParams {
    installationId: number;
    accessTokensUrl: string;
    accountLogin: string;
    accountId: number;
    accountType: string;
  }

  export interface GetInstallationsResponse {
    totalCount: number;
    installations: [GetInstallationsResponse];
  }

  export interface GetMembersResponse {
    id: number;
    login: string;
    avatar_url: string;
  }

  export interface GetForOrgResponse {
    totalCount: number;
    repos: [GetForOrgResponse];
  }

  export interface GetBranchesResponse {
    name: string;
  }
  