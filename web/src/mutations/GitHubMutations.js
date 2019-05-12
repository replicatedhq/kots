import gql from "graphql-tag";

export const createGithubNonce = gql`
  mutation {
    createGithubNonce
  }
`;

export const createGithubAuthToken = gql`
  mutation createGithubAuthToken($code: String!, $state: String!) {
    createGithubAuthToken(code: $code, state: $state) {
      access_token
    }
  }
`;

export const setUpWatchPullRequests = gql`
  mutation setUpWatchPullRequests($id: String!, $org: String!, $target: String!) {
    setUpWatchPullRequests(id: $id, org: $org, target: $target) {
      id
      githubInstallationId
      stateJSON
      watchName
      watchIcon
      lastUpdated
      createdOn
    }
  }
`;

export const logout = gql`
  mutation {
    logout
  }
`;

export const updatePullRequestHistory = gql`
  mutation updatePullRequestHistory($notificationId: String!) {
    updatePullRequestHistory(notificationId: $notificationId) {
      title
      status
      createdOn
      number
      uri
    }
  }
`;
