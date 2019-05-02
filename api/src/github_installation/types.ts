const GQLGetInstallationsResponse = `
type GetInstallationsResponse {
  totalCount: Int
  installations: [GetInstallationsResponseItem]
}
`;

const GQLGetInstallationsResponseItem = `
type GetInstallationsResponseItem {
  login: String
  id: Int
  url: String
  avatar_url: String
}
`;

const GQLGetForOrgResponse = `
type GetForOrgResponse {
  totalCount: Int
  repos: [GetForOrgResponseItem]
}
`;

const GQLGetForOrgResponseItem = `
type GetForOrgResponseItem {
  id: Int
  node_id: String
  url: String
  repository_url: String
  html_url: String
  title: String
  body: String
  created_at: String
  updated_at: String
  name: String
  full_name: String
  default_branch: String
}
`;

const GQLGetBranchesResponseItem = `
type GetBranchesResponseItem {
  name: String
}
`;

const GQLGithubUser = `
type GithubUser {
  login: String
  avatar_url: String
  id: Int
  email: String
}
`;

const GQLGetMembersResponseItem = `
type GetMembersResponseItem {
  id: Int
  login: String
  avatar_url: String
}
`;

const GitHubIntegration = `
type GitHubIntegration {
  installApp: String
  installations: [GitHubInstallation]
}
`;

const GitHubFile = `
type GitHubFile {
  isLoggedIn: Boolean!
  fileContents: String
}
`;

const GitHubInstallation = `
type GitHubInstallation {
  id: String,
  name: String!
  repos: [GitHubRepo],
  accountLogin: String,
  createdAt: String
}
`;

const GitHubRepo = `
type GitHubRepo {
  name: String!
  fullName: String!
}
`;

const GitHubRef = `
type GitHubRef {
  owner: String!
  repoFullName: String!
  branch: String!
  path: String!
}
`;

const GitHubRefInput = `
input GitHubRefInput {
  owner: String!
  repoFullName: String!
  branch: String!
  path: String!
}
`;

export const types = [GitHubIntegration, GitHubInstallation, GitHubRepo, GitHubRef, GitHubRefInput, GitHubFile];

export const vendor = [GQLGetInstallationsResponse, GQLGetInstallationsResponseItem, GQLGetForOrgResponse, GQLGetForOrgResponseItem, GQLGetBranchesResponseItem, GQLGithubUser, GQLGetMembersResponseItem];
