import gql from "graphql-tag";

export const githubUserOrgs = gql`
query installationOrganizations($page: Int) {
  installationOrganizations(page: $page) {
    totalCount
    installations {
      login,
      id,
      url,
      avatar_url
    }
  }
}
`;

export const githubOrgRepos = gql`
query orgRepos($org: String!, $page: Int) {
  orgRepos(org: $org, page: $page) {
    totalCount,
    repos {
      id,
      node_id,
      url,
      repository_url,
      html_url,
      title,
      body,
      created_at,
      updated_at,
      name,
      full_name,
      default_branch
    }
  }
}
`;

export const githubRepoBranches = gql`
query repoBranches($owner: String!, $repo: String!) {
  repoBranches(owner: $owner, repo: $repo) {
    name
  }
}`

export const getOrgMembers = gql`
  query orgMembers($org: String!, $page: Int) {
    orgMembers(org: $org, page: $page) {
      id
      login
      avatar_url
    }
  }
`;

export const validateUpstreamURL = gql`
 query validateUpstreamURL($upstream: String!) {
   validateUpstreamURL(upstream: $upstream)
 }
`;

export const getGitHubInstallationId = gql`
  query {
    getGitHubInstallationId
  }
`;
